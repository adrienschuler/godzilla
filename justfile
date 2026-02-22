mod gateway 'services/gateway'
mod accounts 'services/accounts'
mod presence 'services/presence'
mod chat 'services/chat'
mod history 'services/history'

namespace := "godzilla"

# --- Top-level targets ---

default: unit-test deploy integration-test

deploy: build namespace deploy-services wait seed-deploy status

# --- Minikube ---

port-forward:
    kubectl port-forward -n {{namespace}} svc/gateway-svc 8080:80 &
    kubectl port-forward -n {{namespace}} svc/presence-svc 50051:50051 &
    kubectl port-forward -n {{namespace}} svc/redis-service 6379:6379 &
    kubectl port-forward -n {{namespace}} svc/mongodb 27017:27017 &
    kubectl port-forward -n {{namespace}} svc/history-svc 8000:8000 &
    @echo "Forwarding gateway → localhost:8080, presence → localhost:50051"
    @echo "Forwarding redis → localhost:6379, mongodb → localhost:27017"
    @echo "Forwarding history → localhost:8000"
    @wait

# --- Build targets ---

build:
    just gateway build
    just accounts build
    just presence build
    just chat build
    just history build

# --- Kubernetes targets ---

namespace:
    kubectl create namespace {{namespace}} --dry-run=client -o yaml | kubectl apply -f -
    kubectl config set-context --current --namespace={{namespace}}

redis:
    kubectl apply -f k8s/redis.yaml

mongodb:
    kubectl apply -f k8s/mongodb.yaml

deploy-services: redis mongodb
    just accounts deploy
    just presence deploy
    just chat deploy
    just history deploy
    just gateway deploy

# --- Status / wait ---

wait:
    @echo "Waiting for deployments to be ready..."
    kubectl rollout status deployment/redis --timeout=60s
    kubectl rollout status deployment/mongodb --timeout=60s
    kubectl rollout status deployment/accounts --timeout=60s
    kubectl rollout status deployment/presence --timeout=60s
    kubectl rollout status deployment/chat --timeout=60s
    kubectl rollout status deployment/history --timeout=60s
    kubectl rollout status deployment/gateway --timeout=60s

status:
    @echo ""
    @echo "Pods:"
    kubectl get pods
    @echo ""
    @echo "Services:"
    kubectl get services

# --- Tests ---

unit-test:
    just gateway test
    just accounts test
    just presence test
    just chat test
    just history test

integration-test:
    uv run --project tests pytest tests/test_auth_flow.py -v --tb=short

test: unit-test integration-test

# --- Seed ---

seed:
    uv run --with pymongo --with bcrypt --with dnspython scripts/seed.py

seed-build:
    docker build -t mongodb-seed:latest -f scripts/Dockerfile.seed .

seed-deploy: seed-build
    -kubectl delete job mongodb-seed -n {{namespace}} --ignore-not-found=true
    kubectl apply -f k8s/jobs/seed-job.yaml
    kubectl wait --for=condition=complete job/mongodb-seed -n {{namespace}} --timeout=60s
    kubectl logs job/mongodb-seed -n {{namespace}}

# --- Cleanup ---

clean:
    kubectl delete namespace {{namespace}} --ignore-not-found=true

# --- GKE ---

region := "europe-west1"
registry := region + "-docker.pkg.dev"
project_id := `grep 'project_id' terraform/terraform.tfvars | sed 's/.*= *"\(.*\)"/\1/'`
repo := registry + "/" + project_id + "/godzilla"

gke-setup:
    cd terraform && terraform init && terraform apply
    gcloud container clusters get-credentials godzilla --zone europe-west1-b --project {{project_id}}

gke-auth:
    gcloud auth configure-docker {{registry}}

gke-push: gke-auth
    docker buildx build --platform linux/amd64 --push -t {{repo}}/gateway:latest services/gateway/
    docker buildx build --platform linux/amd64 --push -t {{repo}}/accounts:latest services/accounts/
    docker buildx build --platform linux/amd64 --push -t {{repo}}/presence:latest services/presence/
    docker buildx build --platform linux/amd64 --push -t {{repo}}/chat:latest -f services/chat/Dockerfile .
    docker buildx build --platform linux/amd64 --push -t {{repo}}/history:latest services/history/
    docker buildx build --platform linux/amd64 --push -t {{repo}}/mongodb-seed:latest -f scripts/Dockerfile.seed .

gke-deploy: unit-test namespace
    for f in k8s/*.yaml; do printf '\n---\n'; cat "$f"; done | sed 's|image: \(.*\):latest|image: {{repo}}/\1:latest|' | kubectl apply -f -
    @echo "Waiting for deployments to be ready..."
    kubectl rollout status deployment/mongodb -n {{namespace}} --timeout=120s
    kubectl rollout status deployment/accounts -n {{namespace}} --timeout=120s
    kubectl rollout status deployment/presence -n {{namespace}} --timeout=120s
    kubectl rollout status deployment/chat -n {{namespace}} --timeout=120s
    kubectl rollout status deployment/history -n {{namespace}} --timeout=120s
    kubectl rollout status deployment/gateway -n {{namespace}} --timeout=120s
    -kubectl delete job mongodb-seed -n {{namespace}} --ignore-not-found=true
    sed 's|image: mongodb-seed:latest|image: {{repo}}/mongodb-seed:latest|' k8s/jobs/seed-job.yaml | kubectl apply -f -
    kubectl wait --for=condition=complete job/mongodb-seed -n {{namespace}} --timeout=60s
    kubectl logs job/mongodb-seed -n {{namespace}}
    just integration-test

gke: gke-setup gke-push gke-deploy

gke-teardown:
    cd terraform && terraform destroy
