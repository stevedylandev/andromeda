import { defineConfig } from 'vocs'

export default defineConfig({
  title: 'Andromeda',
  markdown: {
    code: {
      themes: {
        light: "github-dark-high-contrast",
        dark: "github-dark-high-contrast",
      },
    },
  },
  theme: {
    colorScheme: "dark",
    accentColor: "#FFB757",
  },
  sidebar: [
    {
      text: 'Intro',
      items: [
        {
          text: 'Quickstart',
          link: '/quickstart',
        },
        {
          text: 'What is Andromeda',
          link: '/what-is-andromeda',
        },
      ],
    },
    {
      text: 'Apps',
      items: [
        {
          text: 'Feeds',
          link: '/apps/feeds',
        },
        {
          text: 'Jotts',
          link: '/apps/jotts',
        },
        {
          text: 'Sipp',
          link: '/apps/sipp',
        },
        {
          text: 'OG',
          link: '/apps/og',
        },
        {
          text: 'Shrink',
          link: '/apps/shrink',
        },
        {
          text: 'Parcels',
          link: '/apps/parcels',
        },
      ],
    },
    {
      text: 'DIY',
      items: [
        {
          text: 'Stack',
          link: '/diy/stack',
        },
        {
          text: 'Skills',
          link: '/diy/skills',
        },
      ],
    },
  ],
})
