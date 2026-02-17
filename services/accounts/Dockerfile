FROM ruby:3.3-alpine AS build
RUN apk add --no-cache build-base
WORKDIR /app
COPY Gemfile Gemfile.lock* ./
RUN bundle config set without 'development test' && bundle install

FROM ruby:3.3-alpine
RUN apk add --no-cache libgcc libstdc++
WORKDIR /app
COPY --from=build /usr/local/bundle /usr/local/bundle
COPY . .
EXPOSE 8081
CMD ["bundle", "exec", "puma", "-p", "8081", "-e", "production"]
