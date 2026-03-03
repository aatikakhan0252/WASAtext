import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
    plugins: [vue()],
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url))
        }
    },
    define: {
        // When building for production with "yarn build" or "npm run build",
        // the __API_URL__ will be replaced at build time.
        // Default to empty string so that it uses the same origin (relative URLs)
        // when serving from the Go backend.
        // To point to a different backend, set __API_URL__ in the environment:
        //   VITE_API_URL=http://localhost:3000 yarn build
        __API_URL__: JSON.stringify(process.env.VITE_API_URL ?? 'http://localhost:3000'),
    }
}));
