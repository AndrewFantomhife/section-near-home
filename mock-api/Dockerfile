FROM node:18-alpine
WORKDIR /app
RUN npm install -g json-server
EXPOSE 3001
CMD ["json-server", "--watch", "db.json", "--host", "0.0.0.0", "--port", "3001"]