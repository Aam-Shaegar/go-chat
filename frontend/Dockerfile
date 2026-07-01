FROM node:24-alpine AS builder

WORKDIR /frontend

COPY frontend/package*.json ./

RUN npm ci

COPY frontend .

RUN npm run build

FROM nginx:1.29-alpine

COPY docker/nginx/nginx.conf /etc/nginx/conf.d/default.conf

COPY --from=builder /frontend/dist /usr/share/nginx/html