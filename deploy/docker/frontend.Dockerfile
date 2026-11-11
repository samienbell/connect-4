# Build context: repository root
#   docker build -f deploy/docker/frontend.Dockerfile -t connect4-frontend .

FROM node:20-alpine AS build
WORKDIR /app
# TODO: switch to `npm ci` once package-lock.json is committed.
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ .
RUN npm run build

# Unprivileged nginx listens on 8080 and runs as UID 101.
FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY deploy/docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
