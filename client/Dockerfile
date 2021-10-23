FROM node:14.17.1
EXPOSE 3000:3000
WORKDIR /usr/app
COPY bin ./bin
COPY public ./public
COPY routes ./routes
COPY views ./views
COPY app.js .
COPY package.json .
RUN npm install
CMD ["node", "./bin/www"]