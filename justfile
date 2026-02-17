mod gateway 'services/gateway'
mod accounts 'services/accounts'
mod presence 'services/presence'
mod chat 'services/chat'

namespace := "godzilla"

# --- Top-level targets ---

default: unit-test deploy integration-test

deploy: build namespace deploy-services wait status

# --- Minikube ---

port-forward:
    kubectl port-forward -n {{namespace}} svc/gateway-svc 8080:80 &
    kubectl port-forward -n {{namespace}} svc/presence-svc 50051:50051 &
    kubectl port-forward -n {{namespace}} svc/redis-service 6379:6379 &
    kubectl port-forward -n {{namespace}} svc/mongodb-service 27017:27017 &
    @echo "Forwarding gateway → localhost:8080, presence → localhost:50051"
    @echo "Forwarding redis → localhost:6379, mongodb → localhost:27017"
    @wait

# --- Build targets ---

build:
    just gateway build
    just accounts build
    just presence build
    just chat build

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
    just gateway deploy

# --- Status / wait ---

wait:
    @echo "Waiting for deployments to be ready..."
    kubectl rollout status deployment/redis --timeout=60s
    kubectl rollout status deployment/mongodb --timeout=60s
    kubectl rollout status deployment/accounts --timeout=60s
    kubectl rollout status deployment/presence --timeout=60s
    kubectl rollout status deployment/chat --timeout=60s
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

integration-test:
    uv run --project tests pytest tests/test_auth_flow.py -v --tb=short

test: unit-test integration-test

# --- Cleanup ---

clean:
    kubectl delete namespace {{namespace}} --ignore-not-found=true
