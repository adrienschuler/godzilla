ENV["RACK_ENV"] = "test"

require "minitest/autorun"
require "rack/test"
require "mock_redis"
require "json"

# Stub MongoDB collection for tests
class FakeCollection
  def initialize
    @docs = []
    @has_unique_index = false
  end

  def indexes
    self
  end

  def create_one(_keys, **_opts)
    @has_unique_index = true
  end

  def insert_one(doc)
    if @has_unique_index && @docs.any? { |d| d["username"] == doc[:username] }
      error = Mongo::Error::OperationFailure.new("duplicate key error", nil,
        code: 11_000, code_name: "DuplicateKey")
      raise error
    end
    @docs << { "username" => doc[:username], "password" => doc[:password] }
  end

  def find(query)
    results = @docs.select { |d| d["username"] == query[:username] }
    results
  end

  def delete_many(_filter = {})
    @docs.clear
  end
end

class FakeMongoClient
  def database
    self
  end

  def command(**_opts)
    true
  end
end

require_relative "../app"

class AppTest < Minitest::Test
  include Rack::Test::Methods

  def app
    Sinatra::Application
  end

  def setup
    @users = FakeCollection.new
    @users.indexes.create_one({ username: 1 }, unique: true)
    @redis = MockRedis.new

    app.set :users, @users
    app.set :redis, @redis
    app.set :mongo_client, FakeMongoClient.new
  end

  def json_response
    JSON.parse(last_response.body)
  end

  def post_json(path, body)
    post path, body.to_json, "CONTENT_TYPE" => "application/json"
  end

  # --- /healthz ---

  def test_healthz
    get "/healthz"
    assert_equal 200, last_response.status
    assert_equal "ok", json_response["status"]
  end

  # --- /user/register ---

  def test_register_success
    post_json "/user/register", { username: "alice", password: "secret" }
    assert_equal 201, last_response.status
    assert_equal "success", json_response["status"]
  end

  def test_register_duplicate
    post_json "/user/register", { username: "alice", password: "secret" }
    assert_equal 201, last_response.status

    post_json "/user/register", { username: "alice", password: "other" }
    assert_equal 409, last_response.status
    assert_includes json_response["message"], "already exists"
  end

  def test_register_missing_fields
    post_json "/user/register", { username: "", password: "" }
    assert_equal 400, last_response.status
  end

  def test_register_no_body
    post_json "/user/register", {}
    assert_equal 400, last_response.status
  end

  # --- /user/login ---

  def test_login_success
    post_json "/user/register", { username: "bob", password: "pass123" }
    assert_equal 201, last_response.status

    post_json "/user/login", { username: "bob", password: "pass123" }
    assert_equal 200, last_response.status
    assert_equal "success", json_response["status"]

    cookie = last_response.headers["set-cookie"]
    assert_includes cookie, "auth_token="
  end

  def test_login_wrong_password
    post_json "/user/register", { username: "bob", password: "pass123" }
    post_json "/user/login", { username: "bob", password: "wrong" }
    assert_equal 401, last_response.status
  end

  def test_login_nonexistent_user
    post_json "/user/login", { username: "nobody", password: "pass" }
    assert_equal 401, last_response.status
  end

  def test_login_missing_fields
    post_json "/user/login", { username: "", password: "" }
    assert_equal 400, last_response.status
  end

  # --- /user/logout ---

  def test_logout
    post "/user/logout"
    assert_equal 200, last_response.status
    assert_equal "success", json_response["status"]
  end

  def test_logout_clears_session
    post_json "/user/register", { username: "carol", password: "pass" }
    post_json "/user/login", { username: "carol", password: "pass" }

    cookie = last_response.headers["set-cookie"]
    token = cookie[/auth_token=([^;]+)/, 1]
    assert @redis.exists?("session:#{token}")

    set_cookie "auth_token=#{token}"
    post "/user/logout"
    assert_equal 200, last_response.status
    refute @redis.exists?("session:#{token}")
  end

end
