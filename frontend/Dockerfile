FROM node:20 AS build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install || npm install --no-package-lock
COPY frontend/. ./
RUN npm run build

FROM nginx:1.25
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
