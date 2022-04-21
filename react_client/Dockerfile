FROM node:14.17.1
RUN apt-get update
EXPOSE 8080:8080
RUN npm -y install -g typescript
RUN npm -y install -g webpack webpack-cli
WORKDIR /usr/src/app
COPY client ./client
COPY server ./server
WORKDIR /usr/src/app/client
RUN npm -y install
RUN webpack
WORKDIR /usr/src/app/server
RUN npm -y install
CMD ["node", "server.js"]