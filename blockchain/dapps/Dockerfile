FROM node:22-slim

WORKDIR /usr/app

RUN apt-get update && apt-get install -y \
    python3 \
    build-essential

COPY ./blockchain/dapps .

RUN npm install --include=dev

ENTRYPOINT ["npm", "run"]
