import {defineClientAppEnhance} from "@vuepress/client";
import GAConsent from './components/GAConsent.vue';
import { createGtm } from '@gtm-support/vue-gtm';

export default defineClientAppEnhance(({ app, router, siteData}) => {
  app.use(
    createGtm({
      id: __GTM_ID__,
      enabled: false, // disabled by default, will be enabled if user gives consent, see /components/GAConsent.vue
    })
  );
  app.component('GAConsent', GAConsent);
});