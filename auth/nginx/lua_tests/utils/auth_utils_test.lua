-- auth_utils_test.lua
describe("auth_utils", function()
    local auth_utils
    local saved_ngx, saved_config
    local accept_query_token

    -- A request with the given Authorization header, query arguments as nginx
    -- parsed them for this (sub)request, and the client's original URI
    local function request(req)
        _G.ngx = {
            WARN = "warn",
            log = function() end,
            var = {
                http_authorization = req.authorization,
                request_uri = req.request_uri,
            },
            req = { get_uri_args = function() return req.args or {} end },
            unescape_uri = function(s)
                return (s:gsub("%%(%x%x)", function(hex) return string.char(tonumber(hex, 16)) end))
            end,
        }
    end

    setup(function()
        require("spec_helper")
        saved_config = package.loaded["auth.auth_config"]
        package.loaded["auth.auth_config"] = {
            accepts_query_token = function() return accept_query_token end,
        }
        package.loaded["utils.auth_utils"] = nil
        auth_utils = require("utils.auth_utils")
    end)

    teardown(function()
        package.loaded["auth.auth_config"] = saved_config
        package.loaded["utils.auth_utils"] = nil
    end)

    before_each(function()
        saved_ngx = _G.ngx
        accept_query_token = true
    end)

    after_each(function()
        _G.ngx = saved_ngx
    end)

    describe("with an Authorization header", function()
        it("returns the token of a Bearer header", function()
            request({ authorization = "Bearer abc.def.ghi" })

            assert.are.same({ "abc.def.ghi", "header" }, { auth_utils.extract_jwt_token() })
        end)

        it("prefers the header to a query parameter", function()
            request({ authorization = "Bearer from-header", args = { token = "from-query" } })

            assert.are.same({ "from-header", "header" }, { auth_utils.extract_jwt_token() })
        end)

        it("leaves basic auth credentials to nginx", function()
            request({ authorization = "Basic dXNlcjpwYXNz" })

            assert.is_nil(auth_utils.extract_jwt_token())
        end)

        it("accepts a token of the maximum length", function()
            local token = string.rep("a", 4096)
            request({ authorization = "Bearer " .. token })

            assert.are.equal(token, (auth_utils.extract_jwt_token()))
        end)

        it("rejects a token longer than the maximum", function()
            request({ authorization = "Bearer " .. string.rep("a", 4097) })

            assert.is_nil(auth_utils.extract_jwt_token())
        end)
    end)

    describe("with a token in the query string", function()
        it("returns it from any of the accepted parameters", function()
            for _, name in ipairs({ "token", "jwt", "access_token" }) do
                request({ args = { [name] = "abc.def.ghi" } })

                assert.are.same({ "abc.def.ghi", "query" }, { auth_utils.extract_jwt_token() })
            end
        end)

        it("reads it from the client's URI when the subrequest has no arguments", function()
            request({ request_uri = "/ethereum/mainnet?token=abc.def.ghi" })

            assert.are.same({ "abc.def.ghi", "query" }, { auth_utils.extract_jwt_token() })
        end)

        it("decodes a percent-encoded token", function()
            request({ request_uri = "/ethereum/mainnet?token=abc%2Edef%2Eghi" })

            assert.are.equal("abc.def.ghi", (auth_utils.extract_jwt_token()))
        end)

        it("rejects a token longer than the maximum", function()
            request({ args = { token = string.rep("a", 4097) } })

            assert.is_nil(auth_utils.extract_jwt_token())
        end)

        it("ignores a parameter without a value", function()
            -- nginx parses `?token` and `?token=` as token = true
            for _, uri in ipairs({ "/v1/prices?token", "/v1/prices?token=" }) do
                request({ args = { token = true }, request_uri = uri })

                assert.is_nil(auth_utils.extract_jwt_token())
            end
        end)

        it("takes the next parameter when one has no value", function()
            request({ args = { token = true, jwt = "abc.def.ghi" } })

            assert.are.same({ "abc.def.ghi", "query" }, { auth_utils.extract_jwt_token() })
        end)

        it("returns nothing when query tokens are switched off", function()
            accept_query_token = false
            request({ args = { token = "abc.def.ghi" }, request_uri = "/v1/prices?token=abc.def.ghi" })

            assert.is_nil(auth_utils.extract_jwt_token())
        end)
    end)

    it("still returns a header token when query tokens are switched off", function()
        accept_query_token = false
        request({ authorization = "Bearer abc.def.ghi" })

        assert.are.same({ "abc.def.ghi", "header" }, { auth_utils.extract_jwt_token() })
    end)

    it("returns nothing for a request without a token", function()
        request({})

        assert.is_nil(auth_utils.extract_jwt_token())
    end)
end)
