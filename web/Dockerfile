FROM node:22-alpine AS build
WORKDIR /src
COPY web/ ./
ARG VITE_AI_URL=http://localhost:8095
ARG VITE_MODBUS_URL=http://localhost:8090
ENV VITE_AI_URL=$VITE_AI_URL VITE_MODBUS_URL=$VITE_MODBUS_URL
RUN npm install && npm run build

FROM nginx:1.27-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY web/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
