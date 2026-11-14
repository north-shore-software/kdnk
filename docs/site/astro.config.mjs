import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import { readdirSync } from 'node:fs';

const pages = (dir) =>
  readdirSync(new URL(`../${dir}/`, import.meta.url))
    .filter((f) => f.endsWith('.md'))
    .map((f) => `${dir}/${f.slice(0, -3)}`)
    .sort();

export default defineConfig({
  integrations: [
    starlight({
      title: 'kdnk',
      logo: {
        light: '../../brand/kdnk-wordmark-light.svg',
        dark: '../../brand/kdnk-wordmark-dark.svg',
        replacesTitle: true,
      },
      favicon: '/favicon.svg',
      head: [
        { tag: 'link', attrs: { rel: 'icon', href: '/favicon.ico', sizes: '32x32' } },
        { tag: 'link', attrs: { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' } },
      ],
      sidebar: [
        { label: 'Start here', items: ['quickstart'] },
        { label: 'Guides', items: ['apps', 'server', 'providers'] },
        { label: 'Examples', items: pages('examples') },
        { label: 'API reference', items: pages('reference') },
      ],
    }),
  ],
});
