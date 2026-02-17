FROM openresty/openresty:alpine

COPY nginx.conf /usr/local/openresty/nginx/conf
COPY gateway.lua /usr/local/openresty/lualib/gateway.lua

RUN apk update && apk add perl curl && rm -rf /var/cache/apk/*

RUN opm get bungle/lua-resty-session
RUN opm get ktalebian/lua-resty-cookie
