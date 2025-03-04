<!--
Modal dialog which contains slot "form" and two buttons ("Cancel" and "OK").
Should be used to wrap forms which need to be displayed in modal dialog.
Can use used in tandem with ItemFormBase.js. See KeyForm.vue for example.
-->
<template xmlns:v-slot="http://www.w3.org/1999/XSL/Transform">
  <v-dialog
    v-model="dialog"
    :max-width="maxWidth || 400"
    persistent
    :fullscreen="expandable && fullscreen"
    :transition="false"
    :content-class="'item-dialog item-dialog--' + position"
  >
    <v-card>
      <v-card-title>
        <slot name="title">
          <v-icon v-if="icon" :color="iconColor" class="mr-3">{{ icon }}</v-icon>
          {{ title }}
        </slot>

        <v-spacer></v-spacer>

        <v-btn
          icon
          @click="toggleHelp()"
          class="mr-3"
          :style="{opacity: needHelp ? 1 : 0.3}"
          v-if="helpButton"
        >
          <v-icon>mdi-help-box</v-icon>
        </v-btn>

        <v-btn icon @click="toggleFullscreen()" class="mr-3" v-if="expandable">
          <v-icon>mdi-arrow-{{ fullscreen ? 'collapse' : 'expand' }}</v-icon>
        </v-btn>

        <v-btn icon @click="close()" style="margin-right: -6px;">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </v-card-title>

      <v-card-text
        :class="{
          'pb-0': !hideButtons,
          'pa-0': noBodyPaddings,
        }"
        :style="{minHeight: minContentHeight + 'px'}"
      >
        <slot
          name="form"
          :onSave="onSave"
          :onError="clearFlags"
          :needSave="needSave"
          :needReset="needReset"
          :needHelp="needHelp"
        ></slot>
      </v-card-text>

      <v-card-actions v-if="!hideButtons">
        <v-spacer></v-spacer>

        <v-btn
          color="blue darken-1"
          text
          @click="close()"
        >
          {{ cancelButtonText || $t('cancel') }}
        </v-btn>

        <v-btn
          color="blue darken-1"
          text
          @click="needSave = true"
          v-if="saveButtonText != null"
        >
          {{ saveButtonText }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>
<style lang="scss">
  .item-dialog--top {
    align-self: flex-start;
  }
  .item-dialog--center {
  }
</style>
<script>

import EventBus from '@/event-bus';

export default {
  props: {
    position: String,
    title: String,
    icon: String,
    iconColor: String,
    value: Boolean,
    maxWidth: Number,
    minContentHeight: Number,
    eventName: String,
    hideButtons: Boolean,
    dontCloseOnSave: Boolean,
    cancelButtonText: String,
    saveButtonText: String,
    expandable: Boolean,
    name: {
      type: String,
      default: 'Unnamed',
    },
    helpButton: Boolean,
    noBodyPaddings: Boolean,
  },

  data() {
    return {
      dialog: false,
      needSave: false,
      needReset: false,
      fullscreen: null,
      needHelp: false,
    };
  },

  watch: {
    async dialog(val) {
      this.needReset = val;
      this.$emit('input', val);
      if (val) {
        window.addEventListener('keydown', this.handleEscape);
      } else {
        window.removeEventListener('keydown', this.handleEscape);
      }
    },

    async value(val) {
      this.dialog = val;
    },

    fullscreen(val) {
      if (val) {
        localStorage.setItem(`EditDialog_${this.name}__fullscreen`, '1');
      } else {
        localStorage.removeItem(`EditDialog_${this.name}__fullscreen`);
      }
    },
  },

  created() {
    this.fullscreen = localStorage.getItem(`EditDialog_${this.name}__fullscreen`) === '1';
  },

  methods: {
    toggleHelp() {
      this.needHelp = !this.needHelp;
    },

    onSave(e) {
      if (this.dontCloseOnSave) {
        this.clearFlags();
        return;
      }

      this.close(e);
    },

    toggleFullscreen() {
      this.fullscreen = !this.fullscreen;
    },

    close(e) {
      this.dialog = false;

      this.clearFlags();
      if (e) {
        this.$emit('save', e);
        if (this.eventName) {
          EventBus.$emit(this.eventName, e);
        }
      }
      this.$emit('close');
    },

    clearFlags() {
      this.needSave = false;
      this.needReset = false;
    },

    handleEscape(ev) {
      if (ev.key === 'Escape' && this.dialog !== false) {
        this.close();
      }
    },
  },
};
</script>
