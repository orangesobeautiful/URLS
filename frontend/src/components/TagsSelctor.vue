<template>
  <div class="row no-wrap items-center">
    <q-select
      multiple
      use-chips
      stack-label
      :filled="filled"
      :options="existTags"
      v-model="selectedTags"
      :label="label"
      :rules="rules"
      @update:model-value="selectedTagsUpdate"
    >
    </q-select>
    <div class="row no-wrap" v-if="allowNew">
      <q-space class="q-px-sm" />
      <q-btn
        class="new-btn"
        color="primary"
        icon="add"
        @click="showAddTagDialog = true"
      />
    </div>
  </div>
  <q-dialog v-model="showAddTagDialog">
    <q-card>
      <q-card-section>
        <q-input
          class="q-mt-md"
          outlined
          autofocus
          v-model="newTag"
          label="New Tag"
          :rules="[(val) => newTagValid(val)]"
        />
      </q-card-section>
      <q-card-actions align="right">
        <q-btn
          label="Cancel"
          color="secondary"
          @click="showAddTagDialog = false"
        />
        <q-btn
          label="Add"
          color="primary"
          @click="addNewTag"
          :disable="newTag == '' || newTagValid(newTag) != true"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<style scoped>
.q-select {
  width: 100%;
}

.new-btn {
  height: 35px;
  width: 35px;
}
</style>

<script>
import { defineComponent, defineProps, defineEmits, ref, watch } from "vue";

export default defineComponent({
  name: "TagSelector",
});
</script>

<script setup>
const props = defineProps({
  modelValue: {
    type: Array,
    required: true,
  },
  existTags: {
    type: Array,
    required: true,
  },
  allowNew: {
    type: Boolean,
  },
  filled: {
    type: Boolean,
  },
  label: {
    type: String,
  },
  rules: {
    type: Array,
  },
});

watch(
  () => props.modelValue,
  function (newV, _) {
    selectedTags.value = newV;
  }
);

const emit = defineEmits(["update:modelValue"]);

const selectedTags = ref(props.modelValue);
const showAddTagDialog = ref(false);
const newTag = ref("");
const tagStrMaxLen = 15;

function newTagValid(tag) {
  if (tag.length > tagStrMaxLen) {
    return false || "超過上限";
  }
  return true;
}

function addNewTag() {
  if (!selectedTags.value.includes(newTag.value)) {
    selectedTags.value.push(newTag.value);
    emit("update:modelValue", selectedTags.value);
  }
  showAddTagDialog.value = false;
  newTag.value = "";
}

function selectedTagsUpdate() {
  emit("update:modelValue", selectedTags.value);
}
</script>
