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
      sidebar: [
        { label: 'Start here', items: ['quickstart'] },
        { label: 'Guides', items: ['apps', 'server', 'providers'] },
        { label: 'Examples', items: pages('examples') },
        { label: 'API reference', items: pages('reference') },
      ],
    }),
  ],
});
