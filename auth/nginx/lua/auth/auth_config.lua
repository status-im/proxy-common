local _M = {}
local cjson = require "cjson"
local resolver_utils = require "utils.resolver_utils"

-- Seconds between lookups of the auth service address
local DEFAULT_RESOLVE_INTERVAL = 30

local function without_trailing_slash(url)
    -- A trailing slash would change what proxy_pass sends
    return (url:gsub("/$", ""))
end

local function configured_auth_service_url()
    return os.getenv("GO_AUTH_SERVICE_URL") or "http://auth-service:8081"
end

-- Looks the auth service address up in Lua and remembers the answer. Runs when
-- a worker starts and then on a timer: the auth service may come up after
-- nginx, or come back with another IP.
--
-- A failed lookup leaves the address as it was. nginx proxies to an IP only
-- (its own resolver is not involved), so the last IP found is worth more than
-- the host name.
function _M.resolve_auth_service(premature)
    if premature then
        return
    end

    local custom_dns = os.getenv("CUSTOM_DNS") or "127.0.0.11"  -- Docker DNS
    local resolved_url, err = resolver_utils.resolve_url_with_custom_dns(configured_auth_service_url(), custom_dns)
    if not resolved_url then
        ngx.log(ngx.WARN, "auth_config: Failed to resolve auth service URL, keeping ",
                _M.go_auth_service_url, ". Error: ", err or "unknown")
        return
    end

    local url = without_trailing_slash(resolved_url)
    if url ~= _M.go_auth_service_url then
        ngx.log(ngx.NOTICE, "auth_config: Auth service is at ", url)
    end
    _M.go_auth_service_url = url
end

-- Configuration loading function that will be called by timer
function _M.load_config(premature)
    if premature then
        return
    end
    
    -- Use provided config file path or fallback
    local config_path = os.getenv("AUTH_CONFIG_FILE") or "/app/auth_config.json"

    _M.resolve_auth_service()

    -- Read and parse JSON config file
    local config_data = _M.read_json_config(config_path)
    
    if config_data then
        -- JWT rate limiting configuration
        _M.requests_per_token = config_data.requests_per_token or 100
        
        -- JWT token expiry configuration  
        _M.token_expiry_minutes = config_data.token_expiry_minutes or 10
        
        -- Log the initialized values
        ngx.log(ngx.NOTICE, "auth_config: Loaded from ", config_path)
    else
        -- Fallback to environment variables if JSON config fails
        _M.requests_per_token = 100
        _M.token_expiry_minutes = 10
    end
end

-- Whether an environment variable switches a setting off ("false", "0", "off", "no")
local function is_off(value)
    if not value then
        return false
    end
    value = value:lower()
    return value == "false" or value == "0" or value == "off" or value == "no"
end

-- Initialize configuration using timer
function _M.init()
    -- A token in a query parameter is accepted unless AUTH_ACCEPT_QUERY_TOKEN
    -- switches it off. A proxy whose query string is the request itself (and
    -- its cache key) should switch it off.
    _M.accept_query_token = not is_off(os.getenv("AUTH_ACCEPT_QUERY_TOKEN"))

    -- Set default values immediately to prevent race conditions
    _M.go_auth_service_url = without_trailing_slash(configured_auth_service_url())
    _M.requests_per_token = 100
    _M.token_expiry_minutes = 10
    
    -- Schedule config loading using timer
    local ok, err = ngx.timer.at(0, _M.load_config)
    if not ok then
        ngx.log(ngx.ERR, "auth_config: Failed to create timer: ", err)
    end

    -- Keep looking the auth service address up
    local interval = tonumber(os.getenv("AUTH_RESOLVE_INTERVAL")) or DEFAULT_RESOLVE_INTERVAL
    if interval <= 0 then
        interval = DEFAULT_RESOLVE_INTERVAL
    end
    ok, err = ngx.timer.every(interval, _M.resolve_auth_service)
    if not ok then
        ngx.log(ngx.ERR, "auth_config: Failed to create the resolve timer: ", err)
    end
end

-- Read and parse JSON configuration file
function _M.read_json_config(file_path)
    local file = io.open(file_path, "r")
    if not file then
        ngx.log(ngx.ERR, "auth_config: Could not open config file: ", file_path)
        return nil
    end
    
    local content = file:read("*all")
    file:close()
    
    if not content or content == "" then
        ngx.log(ngx.ERR, "auth_config: Config file is empty: ", file_path)
        return nil
    end
    
    local ok, config_data = pcall(cjson.decode, content)
    if not ok then
        ngx.log(ngx.ERR, "auth_config: Failed to parse JSON config: ", config_data)
        return nil
    end
    
    return config_data
end

-- Getter functions for clean access
function _M.get_requests_per_token()
    return _M.requests_per_token
end

function _M.get_token_expiry_minutes()
    return _M.token_expiry_minutes
end

function _M.get_go_auth_service_url()
    return _M.go_auth_service_url
end

function _M.accepts_query_token()
    return _M.accept_query_token ~= false
end

return _M 
