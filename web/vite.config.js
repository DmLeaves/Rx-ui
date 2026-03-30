import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { resolve } from 'path';
import Components from 'unplugin-vue-components/vite';
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers';
import AutoImport from 'unplugin-auto-import/vite';
export default defineConfig({
    plugins: [
        vue(),
        AutoImport({
            imports: [
                'vue',
                {
                    'naive-ui': ['useDialog', 'useMessage', 'useNotification', 'useLoadingBar']
                }
            ]
        }),
        Components({
            resolvers: [NaiveUiResolver()]
        })
    ],
    resolve: {
        alias: {
            '@': resolve(__dirname, 'src'),
        },
    },
    server: {
        port: 3000,
        proxy: {
            '/api': {
                target: 'http://localhost:54321',
                changeOrigin: true,
            },
        },
    },
    build: {
        outDir: 'dist',
        assetsDir: 'assets',
        // 生产构建时嵌入到 Go 二进制
        emptyOutDir: true,
    },
});
