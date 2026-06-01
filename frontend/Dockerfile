FROM node:20-alpine

WORKDIR /app

# Copy package files and install deps
COPY package.json package-lock.json ./
RUN npm ci

# Copy the rest of the application source
COPY . .

# Start the dev server
CMD ["npm", "run", "dev"]
