FROM node:20-bullseye

# Install required OS dependencies for Playwright browsers
# RUN apt-get update && apt-get install -y wget gnupg ca-certificates fonts-noto-color-emoji libglib2.0-0 libnss3 libatk-bridge2.0-0 libxss1 libasound2 libatk1.0-0 libgtk-3-0 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy package files and install deps (including Playwright browsers)
COPY package.json package-lock.json ./
RUN npm install && npx playwright install --with-deps

# Copy the rest of the application source
COPY . .

# Start the dev server (dependencies already installed)
CMD ["npm", "run", "dev"]
