<template>
  <q-dialog v-model="show" @hide="hideHandle">
    <q-card class="dialog">
      <q-card-section>
        <div class="row justify-between">
          <div class="text-h6">新連結</div>
          <div class="row text-grey">
            本月還可以建立&nbsp;
            <div class="text-orange">
              {{ userStore.normalQuota - userStore.normalUsage }}
            </div>
            &nbsp;個一般連結和&nbsp;
            <div class="text-red">
              {{ userStore.customQuota - userStore.customUsage }}
            </div>
            &nbsp;個自訂連結
          </div>
        </div>
      </q-card-section>
      <q-card-section>
        <q-form @submit="createLink">
          <q-input
            filled
            required
            clearable
            v-model="newLinkInfo.destURL"
            label="Destination"
            :rules="[(val) => destURLVaild(val)]"
          />
          <q-space />
          <q-input
            filled
            clearable
            v-model="newLinkInfo.custom"
            label="Custom Short URL (optional)"
            hint="留空以自動隨機生成短網址"
            :rules="[(val) => customValid(val)]"
          />
          <q-space />
          <div class="colmun">
            <div>UTM</div>
            <div class="row">
              <q-input
                v-for="utmName in utmOptions"
                :key="utmName"
                class="q-pa-md utm-input"
                filled
                dense
                clearable
                v-model="newLinkInfo.utmInfo[utmName]"
                :label="utmName"
                :rules="[(val) => utmValid(val)]"
              />
            </div>
          </div>
          <q-space />
          <q-input
            filled
            clearable
            type="textarea"
            v-model="newLinkInfo.note"
            label="Note (optional)"
            :rules="[(val) => noteValid(val)]"
          />
          <q-space />
          <tags-selctor
            filled
            allowNew
            v-model="newLinkInfo.selectedTags"
            label="Tags"
            :existTags="userStore.tags"
            :rules="[(val) => tagsValid(val)]"
          />
          <q-space />
          <q-card-actions align="right">
            <q-btn label="Cancel" @click="closeDialog" />
            <q-btn
              type="submit"
              label="Create"
              color="light-green"
              :disable="!formatVaild()"
            />
          </q-card-actions>
        </q-form>
      </q-card-section>
      <q-inner-loading :showing="createLoading">
        <q-spinner-gears size="50px" color="primary" />
      </q-inner-loading>
    </q-card>
  </q-dialog>
</template>

<style lang="scss" scoped>
.dialog {
  width: 700px;
  max-width: 90vw;
}
.q-space {
  height: 30px;
}
.utm-input {
  width: 33%;
}
</style>

<script>
import { defineComponent, defineProps, defineEmits, ref, watch } from "vue";
import { useQuasar } from "quasar";

import { api } from "boot/axios";
import { useUserStore } from "stores/user";
import TagsSelctor from "components/TagsSelctor.vue";

export default defineComponent({
  name: "CreateLinkDialog",
  components: {
    "tags-selctor": TagsSelctor,
  },
});
</script>

<script setup>
const props = defineProps({
  modelValue: Boolean,
});

watch(
  () => props.modelValue,
  function (newV, _) {
    show.value = newV;
  }
);

const emit = defineEmits(["update:modelValue", "created"]);

const show = ref(props.modelValue);
const utmOptions = ["source", "medium", "campaign", "term", "content"];
const userStore = useUserStore();

function destURLVaild(url) {
  const destURLRgx = new RegExp(
    "^(https?:\\/\\/)?" + // protocol
      "((([a-z\\d]([a-z\\d-]*[a-z\\d])*)\\.)+[a-z]{2,}|" + // domain name
      "((\\d{1,3}\\.){3}\\d{1,3}))" + // OR ip (v4) address
      "(\\:\\d+)?(\\/[-a-z\\d%_.~+]*)*" + // port and path
      "(\\?[;&a-z\\d%_.~+=-]*)?" + // query string
      "(\\#[-a-z\\d_]*)?$",
    "i"
  );
  return !!destURLRgx.test(url) || "不是有效的網址";
}
function customValid(custom) {
  if (custom == null) {
    return true;
  }
  const customRgx = new RegExp(
    "^([-_a-zA-Z\\d/\\p{sc=Han}])*$", // '-', '_', letters, numbers, 中文
    "u"
  );
  return (
    !!customRgx.test(custom) || "目前只接受 '-', '_', 英文字母, 數字 和 中文"
  );
}
function utmValid(utmField) {
  if (utmField == null) {
    return true;
  }
  return utmField.length < utmMaxLen || "超過上限";
}
function noteValid(note) {
  if (note == null) {
    return true;
  }
  return note.length < noteMaxLen || "超過上限";
}
function tagsValid(tags) {
  return tags.length <= tagsMaxLen || "最多添加 " + tagsMaxLen + " tags";
}

const utmMaxLen = 100;
const noteMaxLen = 100;
const tagsMaxLen = 15;
const newLinkInitInfo = {
  destURL: "",
  custom: "",
  utmInfo: {
    source: "",
    medium: "",
    campaign: "",
    term: "",
    content: "",
  },
  note: "",
  selectedTags: [],
};
const newLinkInfo = ref(Object.assign({}, newLinkInitInfo));

function hideHandle() {
  newLinkInfo.value = Object.assign({}, newLinkInitInfo);
  emit("update:modelValue", false);
}

function formatVaild() {
  if (destURLVaild(newLinkInfo.value.destURL) != true) {
    return false;
  }
  if (customValid(newLinkInfo.value.custom) != true) {
    return false;
  }
  for (let key of utmOptions) {
    if (noteValid(newLinkInfo.value.utmInfo[key]) != true) {
      return false;
    }
  }
  if (noteValid(newLinkInfo.value.note) != true) {
    return false;
  }
  if (tagsValid(newLinkInfo.value.selectedTags) != true) {
    return false;
  }

  return true;
}

const createLoading = ref(false);
const { dialog } = useQuasar();
async function createLink() {
  createLoading.value = true;
  await api
    .post("/api/link/v1/link", {
      dest: newLinkInfo.value.destURL,
      custom: newLinkInfo.value.custom,
      utmInfo: newLinkInfo.value.utmInfo,
      note: newLinkInfo.value.note,
      tags: newLinkInfo.value.selectedTags,
    })
    .then((res) => {
      if (res) {
        userStore.normalUsage++;
        if (newLinkInfo.value.custom != "") {
          userStore.customUsage++;
        }
        emit("created");
        dialog({
          title: "新增成功",
        }).onDismiss(() => {
          closeDialog();
        });
      }
    });
  createLoading.value = false;
}

function closeDialog() {
  show.value = false;
}
</script>
