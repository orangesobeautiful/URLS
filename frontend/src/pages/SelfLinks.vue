<template>
  <div class="page">
    <q-page>
      <div class="column q-pa-md upper-block">
        <div class="column filter-block">
          <div class="row items-center justify-between">
            <div>排序方式</div>
            <q-btn-group outline>
              <q-btn
                class="sort-by-btn"
                label="建立時間"
                :color="sortBy === sortByOpt.CreateAt ? 'yellow' : 'grey-3'"
                :text-color="sortBy === sortByOpt.CreateAt ? 'black' : 'grey-6'"
                @click="
                  curPage = 1;
                  sortBy = sortByOpt.CreateAt;
                  searchHandler();
                "
              />
              <q-btn
                class="sort-by-btn"
                label="點擊數"
                :color="sortBy === sortByOpt.TotalClicks ? 'yellow' : 'grey-3'"
                :text-color="
                  sortBy === sortByOpt.TotalClicks ? 'black' : 'grey-6'
                "
                @click="
                  curPage = 1;
                  sortBy = sortByOpt.TotalClicks;
                  searchHandler();
                "
              />
            </q-btn-group>
          </div>
          <q-space style="height: 20px" />
          <div class="tags-filter">
            <tags-selctor
              v-model="selectedTags"
              label="Tag 塞選"
              :existTags="userStore.tags"
            />
          </div>
          <q-space class="q-py-sm" />
        </div>

        <q-space />
        <div class="row self-end">
          <q-btn
            class="create-btn"
            color="light-green"
            text-color="black"
            label="新建連結"
            @click="showAddLink = true"
          />
        </div>
      </div>
      <q-space class="q-py-sm" />

      <q-list class="row">
        <div
          class="links-block q-pa-sm"
          v-for="link in filteredLinks"
          :key="link.idHex"
        >
          <link-item :link-info="link" v-on:deleted="linkDeleted" />
        </div>

        <q-inner-loading :showing="linkListLoading">
          <q-spinner-gears size="50px" color="primary" />
        </q-inner-loading>
      </q-list>

      <div class="q-pa-sm flex flex-center">
        <q-pagination
          color="primary"
          boundary-numbers
          v-model="curPage"
          @update:model-value="searchHandler"
          :max="Math.ceil(totalNum / pageSize)"
        />
      </div>
    </q-page>

    <create-link-dialog
      v-model="showAddLink"
      v-on:created="created"
    ></create-link-dialog>
  </div>
</template>

<style lang="scss" scoped>
@import "src/css/width.scss";
.q-page {
  background-color: #f1efeb;
}

.upper-block {
  width: 100%;
  background-color: white;
}

.filter-block {
  width: 310px;
}

.sort-by-btn {
  width: 115px;
}

.tags-filter {
  width: 100%;
}

.create-btn {
  height: 20px;
}

.links-block {
  width: 100%;
  height: 100%;
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
import { defineComponent, ref, watch } from "vue";
import qs from "qs";

import { api } from "boot/axios";
import { toRouter } from "boot/to-router";
import { useUserStore } from "stores/user";
import TagsSelctor from "components/TagsSelctor.vue";
import CreateLinkDialog from "components/SelfLinks/CreateLinkDialog.vue";
import LinkItem from "components/SelfLinks/LinkItem.vue";

export default defineComponent({
  name: "LinksPages",
  components: {
    "create-link-dialog": CreateLinkDialog,
    "tags-selctor": TagsSelctor,
    "link-item": LinkItem,
  },
});
</script>

<script setup>
const userStore = useUserStore();

const sortByOpt = {
  CreateAt: "createAt",
  TotalClicks: "totalclicks",
};
const sortBy = ref(sortByOpt.CreateAt);
const reverse = ref(true);
const selectedTags = ref([]);
watch(
  () => selectedTags.value,
  function () {
    searchHandler();
  }
);

const totalNumQueried = ref(false);
const totalNum = ref(0);
const curPage = ref(1);
const pageSize = 15;
const filterOptsInit = {
  tags: [],

  sortBy: sortByOpt.CreateAt,
  reverse: true,
  page: 1,
  pageSize: pageSize,
};
const lastFilterOpts = ref(Object.assign({}, filterOptsInit));

const filteredLinks = ref({});
const linkListLoading = ref(false);
async function searchHandler(force) {
  if (force == null || !force) {
    let optsChanged = false;
    if (lastFilterOpts.value.tags != selectedTags.value) {
      optsChanged = true;
      totalNumQueried.value = false;
    }
    if (lastFilterOpts.value.sortBy != sortBy.value) {
      optsChanged = true;
    }
    if (lastFilterOpts.value.reverse != reverse.value) {
      optsChanged = true;
    }
    if (lastFilterOpts.value.page != curPage.value) {
      optsChanged = true;
    }

    if (totalNumQueried.value && !optsChanged) {
      return;
    }
  }

  lastFilterOpts.value = {
    tags: selectedTags.value,

    sortBy: sortBy.value,
    reverse: reverse.value,
    page: curPage.value,
    pageSize: pageSize,
  };

  if (userStore.id == "") {
    toRouter.SigninPage();
    return;
  }

  linkListLoading.value = true;
  if (!totalNumQueried.value) {
    await api
      .get("/api/link/v1/links/count", {
        params: { user_id_hex: userStore.id, tags: lastFilterOpts.value.tags },
        paramsSerializer: (params) => {
          return qs.stringify(params);
        },
      })
      .then((res) => {
        if (res) {
          totalNumQueried.value = true;
          totalNum.value = res.data["totalNum"];
        }
      });
  }

  await api
    .get("/api/link/v1/links", {
      params: {
        user_id_hex: userStore.id,
        tags: lastFilterOpts.value.tags,
        sort_by: lastFilterOpts.value.sortBy,
        reverse: lastFilterOpts.value.reverse,
        page: lastFilterOpts.value.page,
        page_size: pageSize,
      },
      paramsSerializer: (params) => {
        return qs.stringify(params);
      },
    })
    .then((res) => {
      if (res) {
        filteredLinks.value = res.data["linkInfoList"];
      }
    });
  linkListLoading.value = false;
}
watch(
  () => userStore.dataLoaded,
  function (newV, _) {
    if (newV) {
      searchHandler();
    }
  }
);
if (userStore.dataLoaded) {
  searchHandler();
}

function linkDeleted() {
  totalNumQueried.value = false;
  searchHandler(true);
}

const showAddLink = ref(false);
function created() {
  curPage.value = 1;
  totalNumQueried.value = false;
  searchHandler(true);
}
</script>
