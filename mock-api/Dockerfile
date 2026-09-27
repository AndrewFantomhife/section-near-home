FROM node:20-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install --omit=dev

COPY db.json ./

EXPOSE 3001

CMD ["npm", "start"]