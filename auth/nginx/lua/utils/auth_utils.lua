local auth_config = require("auth.auth_config")

local _M = {}

-- JWT tokens are typically < 2KB
local MAX_TOKEN_LENGTH = 4096

-- Query parameters a token is taken from, in this order
local QUERY_TOKEN_PARAMETERS = { "token", "jwt", "access_token" }

-- Extract JWT token from various sources (Authorization header or query parameter)
function _M.extract_jwt_token()
    -- First, try to get token from Authorization header (existing behavior)
    local auth_header = ngx.var.http_authorization
    if auth_header then
        local auth_type, token = auth_header:match("^(%S+)%s+(.+)$")
        if auth_type == "Bearer" and token then
            -- Validate token length to prevent memory issues
            if #token > MAX_TOKEN_LENGTH then
                ngx.log(ngx.WARN, "Token too long, rejecting: ", #token, " bytes")
                return nil, nil
            end
            return token, "header"
        end
    end
    
    if not auth_config.accepts_query_token() then
        return nil, nil
    end

    -- If no valid Authorization header, try query parameters
    local args = ngx.req.get_uri_args()
    
    -- For auth_request subrequests, also check parent request URI for query params
    local request_uri = ngx.var.request_uri
    if request_uri then
        local query_start = request_uri:find("?")
        if query_start then
            local query_string = request_uri:sub(query_start + 1)
            -- Parse query string manually
            for pair in string.gmatch(query_string, "[^&]+") do
                local key, value = pair:match("([^=]+)=?(.*)")
                if key then
                    -- URL decode the key as well for completeness
                    key = ngx.unescape_uri(key)
                    if value ~= "" then
                        -- URL decode the value to handle JWT tokens with encoded characters
                        args[key] = ngx.unescape_uri(value)
                    else
                        args[key] = true
                    end
                end
            end
        end
    end
    
    -- 'access_token' is the common OAuth2 name
    for _, name in ipairs(QUERY_TOKEN_PARAMETERS) do
        local value = args[name]
        -- A parameter without a value is `true`, a repeated one is a table
        if type(value) == "string" then
            -- Validate token length to prevent memory issues
            if #value > MAX_TOKEN_LENGTH then
                ngx.log(ngx.WARN, "Query ", name, " too long, rejecting: ", #value, " bytes")
                return nil, nil
            end
            return value, "query"
        end
    end

    return nil, nil
end


return _M
