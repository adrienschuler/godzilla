FROM node:20-alpine

WORKDIR /app

COPY services/chat/package*.json ./
RUN npm ci --omit=dev

COPY services/chat/src/ ./src/
COPY proto/ ./proto/

ENV PROTO_DIR=/app/proto

EXPOSE 3000
CMD ["node", "src/server.js"]
