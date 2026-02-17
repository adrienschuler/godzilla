-- Mock ngx global
_G.ngx = {
    status = 0,
    HTTP_UNAUTHORIZED = 401,
    HTTP_INTERNAL_SERVER_ERROR = 500,
    INFO = 6, WARN = 5, ERR = 4,
    null = "\0",
    var = { uri = "/socket.io/" },
    req = { set_header = function() end },
    say = function() end,
    exit = function() end,
    log = function() end,
    re = {
        match = function(subject)
            if subject:match("^[a-zA-Z0-9_%-]+$") and #subject <= 128 then return { subject } end
            return nil
        end,
    },
    header = {},
}

-- Mock redis
local mock_redis = {
    set_timeout = function() end,
    connect = function() return true end,
    auth = function() return true end,
    get = function() return "testuser" end,
    set_keepalive = function() return true end,
}
package.loaded["resty.redis"] = { new = function() return mock_redis end }

-- Mock cookie
local mock_cookie = { get = function() return "valid_token" end }
package.loaded["resty.cookie"] = { new = function() return mock_cookie end }

-- Mock cjson
package.loaded["cjson"] = {
    encode = function(t)
        local p = {}
        for k, v in pairs(t) do p[#p + 1] = '"' .. k .. '":"' .. tostring(v) .. '"' end
        return "{" .. table.concat(p, ",") .. "}"
    end,
}

local gateway = dofile("services/gateway/gateway.lua")

local function reset()
    ngx.status = 0
    mock_redis.connect = function() return true end
    mock_redis.get = function() return "testuser" end
    mock_cookie.get = function() return "valid_token" end
    package.loaded["resty.cookie"].new = function() return mock_cookie end
end

describe("handle_auth", function()
    before_each(reset)

    it("sets user header on valid session", function()
        local hdr = {}
        ngx.req.set_header = function(k, v) hdr.k, hdr.v = k, v end
        gateway:handle_auth()
        assert.are.equal("X-Authenticated-User", hdr.k)
        assert.are.equal("testuser", hdr.v)
    end)

    it("rejects missing cookie", function()
        package.loaded["resty.cookie"].new = function() return nil, "err" end
        gateway:handle_auth()
        assert.are.equal(401, ngx.status)
    end)

    it("rejects invalid token format", function()
        mock_cookie.get = function() return "bad token!" end
        gateway:handle_auth()
        assert.are.equal(401, ngx.status)
    end)

    it("returns 500 on redis failure", function()
        mock_redis.connect = function() return nil, "refused" end
        gateway:handle_auth()
        assert.are.equal(500, ngx.status)
    end)

    it("rejects expired/unknown session", function()
        mock_redis.get = function() return ngx.null end
        gateway:handle_auth()
        assert.are.equal(401, ngx.status)
    end)
end)
