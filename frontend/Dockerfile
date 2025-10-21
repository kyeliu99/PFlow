# syntax=docker/dockerfile:1

FROM node:20 AS build
WORKDIR /app
COPY package.json .
COPY package-lock.json .
RUN npm install || npm install --no-package-lock
COPY . .
RUN npm run build

FROM nginx:1.25
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
