FROM node:22-slim


WORKDIR /usr/app

COPY . .

RUN npm install --include=dev

EXPOSE 8545

ENTRYPOINT ["npm", "run", "start:chain"]
