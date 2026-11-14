-- auth_config_test.lua
describe("auth_config", function()
    local auth_config
    local saved_ngx, saved_getenv, saved_resolver
    local env, temp_files, resolved, dns_asked, resolve_timer

    -- Writes content to a temporary file and points AUTH_CONFIG_FILE at it
    local function with_config_file(content)
        local path = os.tmpname()
        local file = assert(io.open(path, "w"))
        file:write(content)
        file:close()
        temp_files[#temp_files + 1] = path
        env.AUTH_CONFIG_FILE = path
    end

    setup(function()
        require("spec_helper")
        saved_resolver = package.loaded["utils.resolver_utils"]
        -- The resolver answers with whatever the test put in `resolved`
        package.loaded["utils.resolver_utils"] = {
            resolve_url_with_custom_dns = function(url, dns)
                dns_asked = { url = url, dns = dns }
                if resolved then
                    return resolved
                end
                return nil, "DNS resolution failed"
            end,
        }
        package.loaded["auth.auth_config"] = nil
        auth_config = require("auth.auth_config")
    end)

    teardown(function()
        package.loaded["utils.resolver_utils"] = saved_resolver
        package.loaded["auth.auth_config"] = nil
    end)

    before_each(function()
        saved_ngx = _G.ngx
        saved_getenv = os.getenv
        env, temp_files, resolved, dns_asked, resolve_timer = {}, {}, nil, nil, nil
        env.AUTH_CONFIG_FILE = "/nonexistent/auth_config.json"
        os.getenv = function(name) return env[name] end
        _G.ngx = {
            ERR = "error", WARN = "warn", NOTICE = "notice",
            log = function() end,
            -- Runs the timer callback at once, the way nginx does at delay 0
            timer = {
                at = function(_, callback) callback(false) return true end,
                -- Kept for the test to fire: resolve_timer.tick()
                every = function(interval, callback)
                    resolve_timer = { interval = interval, tick = function() callback(false) end }
                    return true
                end,
            },
        }
    end)

    after_each(function()
        _G.ngx = saved_ngx
        os.getenv = saved_getenv
        for _, path in ipairs(temp_files) do
            os.remove(path)
        end
    end)

    describe("limits", function()
        it("come from the config file", function()
            with_config_file('{"requests_per_token": 7, "token_expiry_minutes": 3}')

            auth_config.init()

            assert.are.equal(7, auth_config.get_requests_per_token())
            assert.are.equal(3, auth_config.get_token_expiry_minutes())
        end)

        it("fall back to the defaults for what the file does not set", function()
            with_config_file('{"jwt_secret": "s"}')

            auth_config.init()

            assert.are.equal(100, auth_config.get_requests_per_token())
            assert.are.equal(10, auth_config.get_token_expiry_minutes())
        end)

        it("fall back to the defaults when the file is missing or broken", function()
            auth_config.init()
            assert.are.equal(100, auth_config.get_requests_per_token())

            with_config_file("{not json")
            auth_config.init()
            assert.are.equal(100, auth_config.get_requests_per_token())
            assert.are.equal(10, auth_config.get_token_expiry_minutes())
        end)
    end)

    describe("auth service address", function()
        it("is the address the Lua resolver found", function()
            env.GO_AUTH_SERVICE_URL = "http://auth:9000"
            resolved = "http://172.18.0.4:9000"

            auth_config.init()

            assert.are.equal("http://172.18.0.4:9000", auth_config.get_go_auth_service_url())
        end)

        it("is resolved with the configured DNS server", function()
            env.GO_AUTH_SERVICE_URL = "http://auth:9000"
            env.CUSTOM_DNS = "10.0.0.2"

            auth_config.init()

            assert.are.same({ url = "http://auth:9000", dns = "10.0.0.2" }, dns_asked)
        end)

        it("is resolved with Docker's DNS server by default", function()
            auth_config.init()

            assert.are.same({ url = "http://auth-service:8081", dns = "127.0.0.11" }, dns_asked)
        end)

        it("stays the configured one when it cannot be resolved", function()
            env.GO_AUTH_SERVICE_URL = "http://auth:9000"

            auth_config.init()

            assert.are.equal("http://auth:9000", auth_config.get_go_auth_service_url())
        end)

        it("has no trailing slash", function()
            env.GO_AUTH_SERVICE_URL = "http://auth:9000/"
            resolved = "http://172.18.0.4:9000/"

            auth_config.init()

            assert.are.equal("http://172.18.0.4:9000", auth_config.get_go_auth_service_url())
        end)
    end)

    describe("auth service address, looked up again on a timer", function()
        before_each(function()
            env.GO_AUTH_SERVICE_URL = "http://auth:9000"
        end)

        it("follows the auth service to its new IP", function()
            resolved = "http://172.18.0.3:9000"
            auth_config.init()

            resolved = "http://172.18.0.4:9000"
            resolve_timer.tick()

            assert.are.equal("http://172.18.0.4:9000", auth_config.get_go_auth_service_url())
        end)

        it("keeps the last IP found when a lookup fails", function()
            resolved = "http://172.18.0.3:9000"
            auth_config.init()

            resolved = nil
            resolve_timer.tick()

            assert.are.equal("http://172.18.0.3:9000", auth_config.get_go_auth_service_url())
        end)

        it("finds an auth service that came up after nginx", function()
            auth_config.init()
            assert.are.equal("http://auth:9000", auth_config.get_go_auth_service_url())

            resolved = "http://172.18.0.3:9000"
            resolve_timer.tick()

            assert.are.equal("http://172.18.0.3:9000", auth_config.get_go_auth_service_url())
        end)

        it("runs every 30 seconds by default", function()
            auth_config.init()

            assert.are.equal(30, resolve_timer.interval)
        end)

        it("runs every AUTH_RESOLVE_INTERVAL seconds", function()
            env.AUTH_RESOLVE_INTERVAL = "5"

            auth_config.init()

            assert.are.equal(5, resolve_timer.interval)
        end)

        it("falls back to 30 seconds for an interval that is not a positive number", function()
            for _, value in ipairs({ "0", "-1", "soon" }) do
                env.AUTH_RESOLVE_INTERVAL = value

                auth_config.init()

                assert.are.equal(30, resolve_timer.interval)
            end
        end)
    end)

    describe("query tokens", function()
        it("are accepted by default", function()
            auth_config.init()

            assert.is_true(auth_config.accepts_query_token())
        end)

        it("are refused when AUTH_ACCEPT_QUERY_TOKEN switches them off", function()
            for _, value in ipairs({ "false", "0", "off", "no", "FALSE" }) do
                env.AUTH_ACCEPT_QUERY_TOKEN = value

                auth_config.init()

                assert.is_false(auth_config.accepts_query_token())
            end
        end)

        it("are accepted for any other value", function()
            for _, value in ipairs({ "true", "1", "" }) do
                env.AUTH_ACCEPT_QUERY_TOKEN = value

                auth_config.init()

                assert.is_true(auth_config.accepts_query_token())
            end
        end)
    end)
end)
