FROM node:22-alpine AS build
WORKDIR /src
COPY web/ ./
RUN npm install && npm run build

FROM nginx:1.27-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY web/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
