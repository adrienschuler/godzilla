FROM openresty/openresty:alpine

COPY nginx.conf /usr/local/openresty/nginx/conf
COPY gateway.lua /usr/local/openresty/lualib/gateway.lua

RUN apk add --no-cache perl curl

RUN opm get bungle/lua-resty-session ktalebian/lua-resty-cookie
