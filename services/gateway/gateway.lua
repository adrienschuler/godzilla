local cjson = require "cjson"
local redis = require "resty.redis"
local cookie = require "resty.cookie"

local Gateway = {}
Gateway.__index = Gateway

function Gateway.new()
    return setmetatable({
        redis_host = os.getenv("REDIS_SERVICE_HOST") or "127.0.0.1",
        redis_port = tonumber(os.getenv("REDIS_SERVICE_PORT") or "6379"),
        redis_password = os.getenv("REDIS_PASSWORD"),
        redis_timeout = 1000,
        session_prefix = "session:",
        cookie_name = "auth_token",
    }, Gateway)
end

-- Response helpers

function Gateway:respond_json(status_code, status_msg, message_text, data)
    ngx.status = status_code
    local response = { status = status_msg, message = message_text }
    if data then
        for k, v in pairs(data) do response[k] = v end
    end
    ngx.say(cjson.encode(response))
end

function Gateway:respond_error(status_code, message_text)
    self:respond_json(status_code, "error", message_text)
    return ngx.exit(status_code)
end

function Gateway:respond_ok(message_text, data)
    self:respond_json(200, "success", message_text, data)
end

-- Infrastructure

function Gateway:connect_redis()
    ngx.log(ngx.INFO, "connecting to redis at ", self.redis_host, ":", self.redis_port)
    local red = redis:new()
    red:set_timeout(self.redis_timeout)
    local ok, err = red:connect(self.redis_host, self.redis_port)
    if not ok then
        ngx.log(ngx.ERR, "failed to connect to redis: ", err)
        return self:respond_error(ngx.HTTP_INTERNAL_SERVER_ERROR, "Internal server error.")
    end

    if self.redis_password and self.redis_password ~= "" then
        local ok, err = red:auth(self.redis_password)
        if not ok then
            ngx.log(ngx.ERR, "failed to authenticate with redis: ", err)
            return self:respond_error(ngx.HTTP_INTERNAL_SERVER_ERROR, "Internal server error.")
        end
    end

    ngx.log(ngx.INFO, "connected to redis")
    return red
end

function Gateway:release_redis(red)
    local ok, err = red:set_keepalive(10000, 100)
    if not ok then
        ngx.log(ngx.WARN, "failed to set redis keepalive: ", err)
    end
end

function Gateway:get_auth_token()
    local cookies, err = cookie:new()
    if not cookies or err then
        ngx.log(ngx.ERR, "failed to create cookie parser: ", tostring(err))
        return nil
    end
    local token, err = cookies:get(self.cookie_name)
    if err then
        ngx.log(ngx.ERR, "failed to get cookie '", self.cookie_name, "': ", tostring(err))
        return nil
    end
    return token
end

-- Route handlers

function Gateway:init()
    -- This function is called during the NGINX init phase.
    -- Any future gateway-wide initialization can go here.
end

function Gateway:handle_auth()
    ngx.log(ngx.INFO, "handle_auth called for URI: ", ngx.var.uri)
    local token = self:get_auth_token()
    if not token then
        ngx.log(ngx.WARN, "auth token not found")
        return self:respond_error(ngx.HTTP_UNAUTHORIZED, "Unauthorized")
    end
    ngx.log(ngx.INFO, "auth token found")

    if not ngx.re.match(token, "^[a-zA-Z0-9_-]{1,128}$", "jo") then
        ngx.log(ngx.WARN, "invalid auth token format")
        return self:respond_error(ngx.HTTP_UNAUTHORIZED, "Unauthorized")
    end

    local red = self:connect_redis()
    if not red then
        return -- respond_error is called in connect_redis
    end

    local username, err = red:get(self.session_prefix .. token)
    if err then
        ngx.log(ngx.ERR, "redis GET failed: ", err)
        self:release_redis(red)
        return self:respond_error(ngx.HTTP_INTERNAL_SERVER_ERROR, "Internal server error")
    end

    if not username or username == ngx.null then
        ngx.log(ngx.WARN, "session not found in redis for token")
        self:release_redis(red)
        return self:respond_error(ngx.HTTP_UNAUTHORIZED, "Unauthorized")
    end

    self:release_redis(red)
    ngx.log(ngx.INFO, "session found for ", username)
    ngx.req.set_header("X-Authenticated-User", username)
end

return Gateway.new()
