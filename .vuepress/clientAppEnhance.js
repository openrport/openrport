import {defineClientAppEnhance} from "@vuepress/client";
import GAConsent from './components/GAConsent.vue';
import VueGtag from "vue-gtag-next";

export default defineClientAppEnhance(({ app}) => {
  app.use(VueGtag, {
    isEnabled: false,
    property: { id: __GA4_ID__ },
  });
  app.component('GAConsent', GAConsent);
});