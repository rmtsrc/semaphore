<template>
  <v-form
    ref="form"
    lazy-validation
    v-model="formValid"
    v-if="item != null"
  >
    <v-alert
      :value="formError"
      color="error"
      class="pb-2"
    >{{ formError }}</v-alert>

    <v-text-field
      v-model="item.name"
      :label="$t('name')"
      :rules="[v => !!v || $t('name_required')]"
      required
      :disabled="formSaving"
    ></v-text-field>

    <v-text-field
      v-model="item.webhook"
      :label="$t('webhook')"
      required
      :disabled="formSaving"
    ></v-text-field>

    <v-text-field
      type="number"
      v-model.number="item.max_parallel_tasks"
      :label="$t('maxNumberOfParallelTasksOptional')"
      required
      :disabled="formSaving"
    ></v-text-field>

    <v-checkbox
      v-if="isNew"
      class="mb-2"
      v-model="generatePublicKey"
      :label="$t('Generate the private/public key')"
      :disabled="formSaving"
      hide-details
    ></v-checkbox>

    <v-textarea
      outlined
      class="pb-0 CodeEditor"
      v-model="item.public_key"
      :label="$t('Public Key')"
      :disabled="formSaving || generatePublicKey"
      auto-grow
    ></v-textarea>

    <v-checkbox
      class="mt-0"
      v-model="item.active"
      :label="$t('enabled')"
      :disabled="formSaving"
      hide-details
    ></v-checkbox>
  </v-form>
</template>
<style lang="scss">
.CodeEditor textarea {
  font-family: monospace;
  font-size: 14px;
  white-space: pre;
  overflow-wrap: normal;
  overflow-x: scroll;
  line-height: 1.2;
}
</style>
<script>
import ItemFormBase from '@/components/ItemFormBase';

export default {
  props: {
    isAdmin: Boolean,
  },

  mixins: [ItemFormBase],

  data() {
    return {
      generatePublicKey: null,
    };
  },

  methods: {
    getItemsUrl() {
      return '/api/runners';
    },

    beforeLoadData() {
      this.generatePublicKey = this.isNew;
    },

    beforeSave() {
      if (!this.item.max_parallel_tasks) {
        this.item.max_parallel_tasks = 0;
      }

      if (this.isNew && this.generatePublicKey) {
        this.item.public_key = null;
      }
    },

    getSingleItemUrl() {
      return `/api/runners/${this.itemId}`;
    },
  },
};
</script>
