const fs = require('fs');
const path = require('path');

module.exports = {
  base: '/',
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
        href: `/favicon/favicon-16x16.png`,
      },
    ],
    [
      'link',
      {
        rel: 'icon',
        type: 'image/png',
        sizes: '32x32',
        href: `/favicon/favicon-32x32.png`,
      },
    ],
    ['link', { rel: 'manifest', href: '/favicon/site.webmanifest' }],
    ['meta', { name: 'application-name', content: 'docs' }],
    ['meta', { name: 'apple-mobile-web-app-title', content: 'docs' }],
    [
      'meta',
      { name: 'apple-mobile-web-app-status-bar-style', content: 'black' },
    ],
    [
      'link',
      { rel: 'apple-touch-icon', href: `/favicon/apple-touch-icon.png` },
    ],
    ['meta', { name: 'msapplication-TileColor', content: '#0075ec' }],
    ['meta', { name: 'theme-color', content: '#0075ec' }],
  ],
  editLink: false,

  plugins: [
    // container
    // Docs: https://vuepress2.netlify.app/reference/plugin/container.html
    [
      '@vuepress/container',
      {
        type: 'vimeo',
        validate: (params) => {
          return params.trim().match(/^vimeo\s(.*)$/);
        },
        render: (tokens, index) => {
          if (tokens[index].nesting === 1) {
            const info = tokens[index].info.trim().split(' ');
            // opening tag
            return `<div class="iframe-container">\n<iframe src="${info[1]}?byline=0&portrait=0" allow="autoplay; fullscreen; picture-in-picture" allowfullscreen></iframe>`;
          } else {
            // closing tag
            return '</div>\n';
          }
        }
      },
    ],
  ],

  // additional global constants
  define: {
    __GA4_ID__: 'G-QVHYG93PE3',
  },

  // client app root component files
  clientAppRootComponentFiles: path.resolve(__dirname, './components/GAConsent.vue'),

  themeConfig: {
    contributors: false,
    editLink: false,
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
    repo: 'cloudradar-monitoring/rport',
    repoLabel: 'Github-Repo',
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
