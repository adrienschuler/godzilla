require "sinatra"
require "sinatra/json"
require "mongo"
require "redis"
require "bcrypt"
require "securerandom"
require "json"
require "logger"

BCRYPT_COST    = 12
SESSION_TTL    = 86_400 # 24 hours
COOKIE_NAME    = "auth_token"
SESSION_PREFIX = "session:"

configure do
  set :port, ENV.fetch("PORT", "8081").to_i
  set :bind, "0.0.0.0"
  set :logger, Logger.new($stdout)
  set :logging, true
end

configure :development, :production do
  # MongoDB
  mongo_uri = ENV.fetch("MONGO_URI", "mongodb://localhost:27017")
  db_name   = ENV.fetch("MONGO_DB", "godzilla")
  mongo_client = Mongo::Client.new(mongo_uri, database: db_name)
  users = mongo_client[:users]
  users.indexes.create_one({ username: 1 }, unique: true)
  set :users, users
  set :mongo_client, mongo_client

  # Redis
  redis_host = ENV.fetch("REDIS_SERVICE_SERVICE_HOST", "127.0.0.1")
  redis_port = ENV.fetch("REDIS_SERVICE_SERVICE_PORT", "6379")
  set :redis, Redis.new(host: redis_host, port: redis_port.to_i)

  settings.logger.info("connected to mongodb and redis")
end

before do
  content_type :json
end

helpers do
  def json_body
    @json_body ||= JSON.parse(request.body.read, symbolize_names: true)
  rescue JSON::ParserError
    nil
  end

  def error_response(status_code, message)
    halt status_code, json(status: "error", message: message)
  end
end

# --- Routes ---

get "/healthz" do
  settings.redis.ping
  settings.mongo_client.database.command(ping: 1)
  json status: "ok"
rescue => e
  settings.logger.error("healthz failed: #{e.message}")
  error_response 503, "service unavailable"
end

post "/user/register" do
  body = json_body
  error_response 400, "Username and password are required" unless body && body[:username].to_s != "" && body[:password].to_s != ""

  hashed = BCrypt::Password.create(body[:password], cost: BCRYPT_COST)

  begin
    settings.users.insert_one(username: body[:username], password: hashed.to_s)
  rescue Mongo::Error::OperationFailure => e
    if e.message.include?("duplicate key") || e.code == 11_000
      error_response 409, "Username already exists"
    end
    raise
  end

  settings.logger.info("user registered: #{body[:username]}")
  status 201
  json status: "success", message: "User registered successfully"
end

post "/user/login" do
  body = json_body
  error_response 400, "Username and password are required" unless body && body[:username].to_s != "" && body[:password].to_s != ""

  doc = settings.users.find(username: body[:username]).first
  error_response 401, "Invalid credentials" unless doc

  stored = BCrypt::Password.new(doc["password"])
  error_response 401, "Invalid credentials" unless stored == body[:password]

  session_id = SecureRandom.hex(32)
  settings.redis.setex("#{SESSION_PREFIX}#{session_id}", SESSION_TTL, body[:username])

  response.set_cookie(COOKIE_NAME, {
    value: session_id,
    max_age: SESSION_TTL,
    path: "/",
    httponly: true,
    same_site: :lax
  })

  json status: "success", message: "Logged in successfully"
end

post "/user/logout" do
  token = request.cookies[COOKIE_NAME]
  settings.redis.del("#{SESSION_PREFIX}#{token}") if token

  response.delete_cookie(COOKIE_NAME, path: "/")

  json status: "success", message: "Logged out"
end

