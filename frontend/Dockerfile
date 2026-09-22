FROM node:24-alpine AS build

WORKDIR /src

# npm ci from the lockfile, in its own layer: dependencies change far less often
# than source, and ci is reproducible where install is not.
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# The build script is `vue-tsc -b && vite build`, so a type error fails the image
# build rather than shipping.
RUN npm run build

FROM nginx:1.29-alpine

COPY --from=build /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
