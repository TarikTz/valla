import { defineConfig } from 'vitepress'

export default defineConfig({
  srcExclude: ['superpowers/**', 'node_modules/**'],
  title: 'Valla',
  description: 'Scaffold your full stack in seconds. Zero-config local HTTPS.',
  base: '/valla/',

  head: [
    ['link', { rel: 'icon', href: '/valla/favicon.ico' }],
  ],

  themeConfig: {
    logo: { light: '/logo-light.svg', dark: '/logo-dark.svg', alt: 'Valla' },
    siteTitle: 'Valla',

    nav: [
      { text: 'Scaffold', link: '/scaffold/', activeMatch: '/scaffold/' },
      { text: 'Serve', link: '/serve/', activeMatch: '/serve/' },
      { text: 'Contributing', link: '/contributing' },
      {
        text: 'npm',
        link: 'https://www.npmjs.com/package/valla-cli',
      },
    ],

    sidebar: {
      '/scaffold/': [
        {
          text: 'Scaffold',
          items: [
            { text: 'Getting started', link: '/scaffold/' },
            { text: 'Supported stacks', link: '/scaffold/stacks' },
            { text: 'Output modes', link: '/scaffold/output-modes' },
            { text: 'Workflows', link: '/scaffold/workflows' },
          ],
        },
      ],
      '/serve/': [
        {
          text: 'Secure Serve',
          items: [
            { text: 'What is valla serve?', link: '/serve/' },
            { text: 'Setup (valla trust)', link: '/serve/setup' },
            { text: 'Routing', link: '/serve/routing' },
            { text: 'Dashboard (--ui)', link: '/serve/dashboard' },
            { text: 'Platforms & TLDs', link: '/serve/platforms' },
            { text: 'Flag reference', link: '/serve/reference' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/tariktz/valla' },
      { icon: 'npm', link: 'https://www.npmjs.com/package/valla-cli' },
    ],

    search: {
      provider: 'local',
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024-present Tarik Omercehajic',
    },

    editLink: {
      pattern: 'https://github.com/tariktz/valla/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },
  },
})
