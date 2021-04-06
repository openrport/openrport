const fs = require('fs');
const path = require('path');

module.exports = {
  base: '/rport/',
  lang: 'en-DE',
  title: 'rport',
  description: 'Rport helps you to manage your remote servers without the hassle of VPNs, chained SSH connections, jump-hosts, or the use of commercial tools like TeamViewer and its clones.',
  head: [
    [
      'link',
      {
        rel: 'icon',
        type: 'image/png',
        sizes: '16x16',
        href: `/rport/favicon/favicon-16x16.png`,
      },
    ],
    [
      'link',
      {
        rel: 'icon',
        type: 'image/png',
        sizes: '32x32',
        href: `/rport/favicon/favicon-32x32.png`,
      },
    ],
    ['link', { rel: 'manifest', href: '/rport/favicon/site.webmanifest' }],
    ['meta', { name: 'application-name', content: 'docs' }],
    ['meta', { name: 'apple-mobile-web-app-title', content: 'docs' }],
    [
      'meta',
      { name: 'apple-mobile-web-app-status-bar-style', content: 'black' },
    ],
    [
      'link',
      { rel: 'apple-touch-icon', href: `/rport/favicon/apple-touch-icon.png` },
    ],
    ['meta', { name: 'msapplication-TileColor', content: '#0075ec' }],
    ['meta', { name: 'theme-color', content: '#0075ec' }],
  ],
  editLink: false,

  themeConfig: {
    logo: 'logo/rport-img-text.svg',
    lastUpdated: false,
    navbar: [
      {
        text: 'Documentation',
        link: '/docs/',
      },
      {
        text: 'Help',
        link: 'https://github.com/cloudradar-monitoring/rport/discussions',
      },
      {
        text: 'Wiki',
        link: 'https://github.com/cloudradar-monitoring/rport/wiki',
      },
      {
        text: 'Download',
        link: 'https://github.com/cloudradar-monitoring/rport/releases',
      },
    ],
    sidebar: {
      '/docs/': [
        {
          isGroup: true,
          text: 'Documentation',
          children: getSideBar('docs'),
        },
      ],
    },
  },
};

function getSideBar(folder) {
  const extension = [".md"];

  return fs
    .readdirSync(path.join(`${__dirname}/../${folder}`))
    .filter(
      (item) =>
        //item.toLowerCase() != "readme.md"  &&
        fs.statSync(path.join(`${__dirname}/../${folder}`, item)).isFile() &&
        extension.includes(path.extname(item))
    );
}
