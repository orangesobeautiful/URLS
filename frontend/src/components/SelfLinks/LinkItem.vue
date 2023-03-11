<template>
  <q-card class="item-block" @click="itemClick">
    <q-card-section>
      <div class="row no-wrap justify-between">
        <div class="colmun item-left-block">
          <div class="text-subtitle2 text-orange text-ellipsis">
            {{ linkInfo.short }}
          </div>
          <div class="text-subtitle2 text-grey-7 text-ellipsis" hint>
            {{ linkInfo.fullDest }}
            <q-tooltip>
              {{ linkInfo.fullDest }}
            </q-tooltip>
          </div>
        </div>
        <div class="column text-right">
          <q-space />
          <div class="row justify-center self-end">
            <q-icon name="show_chart" />
            <q-space class="q-px-sm" />
            {{ linkInfo.totalClicks }}
          </div>
          <q-space />
          <div>{{ createDatePlain }}</div>
        </div>
      </div>
    </q-card-section>
  </q-card>

  <q-dialog v-model="showDetailDialog">
    <q-card class="detail-dialog">
      <q-card-section>
        <q-splitter v-model="splitterModel" horizontal>
          <template v-slot:before>
            <q-tabs v-model="detailtab" horizontal class="text-teal">
              <q-tab name="normal" icon="normal" label="基本資訊" />
              <q-tab name="clicks" icon="clicks" label="來源分析" />
            </q-tabs>
          </template>

          <template v-slot:after>
            <q-tab-panels
              v-model="detailtab"
              animated
              horizontal
              draggable="false"
              transition-prev="jump-up"
              transition-next="jump-up"
            >
              <q-tab-panel name="normal">
                <div class="row justify-start q-pb-sm">
                  <div class="text-h4 text-orange short-text">
                    {{ linkInfo.short }}
                  </div>
                  <q-btn
                    flat
                    icon="content_copy"
                    color="orange"
                    size="md"
                    padding="xs"
                    @click="copyShort"
                  ></q-btn>
                </div>
                <div class="row no-wrap items-center justify-start">
                  <q-icon
                    name="keyboard_double_arrow_right"
                    color="grey"
                    class="q-pr-sm"
                  />
                  <div class="text-grey full-dest" hint>
                    {{ linkInfo.fullDest }}
                    <q-tooltip>
                      {{ linkInfo.fullDest }}
                    </q-tooltip>
                  </div>
                </div>
                <div class="colmun text-right">
                  <q-space class="q-pb-sm" />
                  <div>Total Clicks: {{ linkInfo.totalClicks }}</div>
                  <q-space class="q-pb-sm" />
                  <div>
                    {{ createDateDetail }}
                  </div>
                </div>
                <q-space class="q-py-sm" />

                <div class="text-subtitle2">Tags:</div>
                <q-space class="q-py-sm" />
                <tags-selctor
                  filled
                  allowNew
                  v-model="linkTags"
                  :existTags="userStore.tags"
                  :rules="[(val) => tagsValid(val)]"
                  @update:model-value="changeHandle"
                />
                <q-space class="q-py-lg" />
                <div class="text-subtitle2">備註:</div>
                <q-space class="q-py-sm" />
                <q-input
                  filled
                  type="textarea"
                  :input-style="{ resize: 'none' }"
                  v-model="note"
                  :rules="[(val) => noteValid(val)]"
                  @update:model-value="changeHandle"
                />
                <q-space class="q-py-lg" />
                <div class="row justify-between apply-block">
                  <q-btn icon="delete" color="red-5" @click="deleteBtnClick"
                    >刪除</q-btn
                  >
                  <q-btn v-if="showApplyBtn" color="orange" text-color="black"
                    >更新</q-btn
                  >
                </div>
              </q-tab-panel>

              <q-tab-panel name="clicks">
                <div class="text-center text-h5">
                  總點擊數: {{ linkInfo.totalClicks }}
                </div>
                <q-space class="q-py-md" />
                <div class="row" v-if="linkInfo.totalClicks > 0">
                  <div
                    class="click-chart"
                    v-for="clicksData in clicksDataList"
                    :key="clicksData.Title"
                  >
                    <div class="text-h6">
                      {{ clicksData.Title }}
                    </div>
                    <apexchart
                      type="donut"
                      :options="clicksData.ChartData.Options"
                      :series="clicksData.ChartData.Series"
                    />
                  </div>
                </div>
              </q-tab-panel>
            </q-tab-panels>
          </template>
        </q-splitter>
      </q-card-section>
      <q-inner-loading :showing="detailLoading">
        <q-spinner-gears size="50px" color="primary" />
      </q-inner-loading>
    </q-card>
  </q-dialog>
</template>

<style lang="scss" scoped>
@import "src/css/width.scss";
.item-block:hover {
  cursor: pointer;
}
.text-ellipsis {
  overflow: hidden;
  white-space: nowrap;

  text-overflow: ellipsis;
}

.item-left-block {
  width: calc(100% - 100px);
}

.detail-dialog {
  min-width: 80vw;
  height: 723px;
}

.short-text {
  @include xxs-width {
    font-size: 23px;
  }

  @include xs-width {
    font-size: 27px;
  }
}

.apply-block {
  height: 35px;
}

.click-chart {
  @include xs-width {
    width: 100%;
  }
  @include sm-width {
    width: 50%;
  }
  @include md-width {
    width: 50%;
  }
  @include lg-width {
    width: 33%;
  }
  @include xl-width {
    width: 25%;
  }
}
</style>

<script>
import { defineComponent, defineProps, defineEmits, ref } from "vue";
import { useQuasar, copyToClipboard } from "quasar";
import VueApexCharts from "vue3-apexcharts";

import { api } from "boot/axios";
import { useUserStore } from "stores/user";
import TagsSelctor from "components/TagsSelctor.vue";

export default defineComponent({
  name: "LinkItem",
  components: {
    "tags-selctor": TagsSelctor,
    apexchart: VueApexCharts,
  },
});
</script>

<script setup>
const { dialog, notify } = useQuasar();
const userStore = useUserStore();

const props = defineProps({
  linkInfo: null,
});

const emit = defineEmits(["deleted"]);

const createDate = new Date(props.linkInfo.createAt);
const createYear = createDate.getFullYear();
const createMonth = createDate.getMonth() + 1;
const createDay = createDate.getDate();
const createHour = createDate.getHours();
const createMin = createDate.getMinutes();
const createSec = createDate.getSeconds();
let pZero = (val, len) => {
  return String(val).padStart(len, "0");
};

// createDatePlain (2006/01/02)
const createDatePlain = `${createYear}-${pZero(createMonth, 2)}-${pZero(
  createDay,
  2
)}`;
// createDatePlain (2006/01/02 15:04:05)
const createDateDetail = `${createDatePlain} ${createHour}:${pZero(
  createMin,
  2
)}:${pZero(createSec, 2)}`;

const showDetailDialog = ref(false);
const detailLoading = ref(false);

function itemClick() {
  detailtab.value = "normal";
  showDetailDialog.value = true;
}

const detailtab = ref("normal");
const splitterModel = ref(1500);

function copyShort() {
  copyToClipboard(props.linkInfo.short)
    .then(() => {
      // success!
      notify({
        message: "複製成功",
        color: "green",
        timeout: 500,
      });
    })
    .catch(() => {
      // fail
      notify({
        message: "複製失敗",
        color: "red",
        timeout: 500,
      });
    });
}

const tagsMaxLen = 15;

const sortedTags = Array.from(props.linkInfo.tags).sort();
const linkTags = ref(props.linkInfo.tags);
function tagsValid(tags) {
  return tags.length <= tagsMaxLen || "最多添加 " + tagsMaxLen + " tags";
}

const noteMaxLen = 100;
const note = ref(props.linkInfo.note);
function noteValid(note) {
  if (note == null) {
    return true;
  }
  return note.length < noteMaxLen || "超過上限";
}

const showApplyBtn = ref(false);
function changeHandle() {
  const arrEquals = (a, b) => {
    return a.length === b.length && a.every((v, i) => v === b[i]);
  };

  let linkTagsSorted = Array.from(linkTags.value).sort();
  if (!arrEquals(linkTagsSorted, sortedTags)) {
    showApplyBtn.value = true;
    return;
  }
  if (note.value != props.linkInfo.note) {
    showApplyBtn.value = true;
    return;
  }

  showApplyBtn.value = false;
}

async function deleteBtnClick() {
  dialog({
    title: "刪除確認",
    message: "確定要刪除此連結嗎?",
    cancel: true,
    focus: "cancel",
  }).onOk(async () => {
    detailLoading.value = true;
    await api
      .delete("/api/link/v1/link/" + props.linkInfo.idHex)
      .then((res) => {
        if (res) {
          dialog({
            title: "刪除成功",
          });
          showDetailDialog.value = false;
          emit("deleted");
        }
      });
    detailLoading.value = false;
  });
}

function mapToChartData(v) {
  let labels = [];
  let series = [];
  for (const key in v) {
    labels.push(key);
    series.push(parseInt(v[key]));
  }

  return {
    Options: {
      labels: labels,
    },
    Series: series,
  };
}

const clicksTitle = ["國家", "裝置", "瀏覽器", "作業系統"];
const clicksMapKey = [
  "countryClicks",
  "deviceClicks",
  "browserClicks",
  "osClicks",
];
const clicksDataList = [];
for (let i = 0; i < clicksTitle.length; i++) {
  clicksDataList.push({
    Title: clicksTitle[i],
    ChartData: mapToChartData(props.linkInfo[clicksMapKey[i]]),
  });
}
</script>
