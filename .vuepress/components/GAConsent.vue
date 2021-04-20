<template>
  <div class="ga-consent" :class="{ 'ga-consent--show' : show }">
    <p>
      By clicking "Accept", you agree that we collect measurement data with Google Analytics.
    </p>
    <button type="button" class="button primary" @click="enableGA">Accept</button>
  </div>
</template>

<script>
import { defineComponent } from 'vue';
import { useState } from "vue-gtag-next";

export default defineComponent({
  name: 'GAConsent',

  data () {
    return {
      localStorageItemKey: 'rport_GA',
      acceptedGA: false,
      show: false,
    }
  },

  mounted () {
    if (process.env.NODE_ENV === 'development') {
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
      const { isEnabled } = useState();

      isEnabled.value = true;
      localStorage.setItem(this.localStorageItemKey, 'accepted');
      this.acceptedGA = true;
      this.show = false;
    },
    disableGA() {
      const { isEnabled } = useState();

      isEnabled.value = false;
      localStorage.removeItem(this.localStorageItemKey);
      this.acceptedGA = false;
      this.show = true;
    },
  },

})
</script>