<template>
  <div class="object-refs-view integration-refs-view">
    <v-alert
      type="warning"
    >
      The {{ objectTitle }} can't be deleted because it used by the resources below
    </v-alert>
    <div
      v-for="s in sections"
      class="object-refs-view__section"
      :key="s.slug"
    >
      <div class="object-refs-view__section-title">
        <v-icon small class="mr-2">mdi-{{ s.icon }}</v-icon>{{ s.title }}:
      </div>

      <div class="ml-6">
        <span v-for="t in integrationRefs[s.slug]" class="object-refs-view__link-wrap" :key="t.id">
          <router-link
            :to="`/project/${projectId}/integrations/${t.id}`"
            class="object-refs-view__link">{{ t.name }}</router-link>
        </span>
      </div>
    </div>
  </div>
</template>
