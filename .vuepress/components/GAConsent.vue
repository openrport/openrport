<template>
  <div class="ga-consent" :class="{ 'ga-consent--show' : show }">
    <p>
      By clicking "Accept", you agree that we store Cookies from Google to provide services and analyse traffic.
    </p>
    <button type="button" class="button primary" @click="enableGA">Accept</button>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'GAConsent',

  data () {
    return {
      localStorageItemKey: __GTM_LS_ITEM__,
      show: false,
    }
  },

  mounted () {
    if (__DEV__) {
      return;
    }
    if (localStorage.getItem(this.localStorageItemKey)) {
      this.enableGA();
    } else {
      this.disableGA();
    }
  },

  methods: {
    enableGA() {
      this.$gtm.enable(true);
      localStorage.setItem(this.localStorageItemKey, 'accepted');
      this.show = false;
    },
    disableGA() {
      this.$gtm.enable(false);
      localStorage.removeItem(this.localStorageItemKey);
      this.show = true;
    },
  },
})
</script>