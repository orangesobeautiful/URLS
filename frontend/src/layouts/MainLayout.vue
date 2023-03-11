<template>
  <q-layout view="hHh lpR fFf">
    <q-header elevated>
      <q-toolbar>
        <q-toolbar-title class="text-black">
          <q-btn flat @click="toRouter.HomePage()"> URLS </q-btn>
        </q-toolbar-title>

        <div class="row" v-if="userStore.dataLoaded">
          <div v-if="!userStore.hasLogin">
            <q-btn @click="toRouter.RegisterPage()">Register</q-btn>
            <q-btn @click="toRouter.SigninPage()">Signin</q-btn>
          </div>
          <div v-if="userStore.hasLogin" class="row items-center">
            <div class="row">{{ userStore.showName }}</div>
            <q-avatar class="q-ml-sm">
              <img :src="avatar" />
              <q-menu :style="{ backgroundColor: '#eee', color: 'blue' }">
                <q-list style="min-width: 100px">
                  <q-item clickable @click="toRouter.SelfLinksPage()">
                    <q-item-section>我的連結</q-item-section>
                  </q-item>
                  <q-separator />
                  <q-item
                    v-if="userStore.isManager"
                    clickable
                    @click="toRouter.DashboardPage()"
                  >
                    <q-item-section>控制台</q-item-section>
                  </q-item>
                  <q-separator />
                  <q-item clickable @click="logout">
                    <q-item-section>登出</q-item-section>
                  </q-item>
                </q-list>
              </q-menu>
            </q-avatar>
          </div>
        </div>
      </q-toolbar>
    </q-header>

    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script>
import { defineComponent, onBeforeMount } from "vue";

import { api } from "boot/axios";
import { toRouter } from "boot/to-router";
import { useUserStore } from "stores/user";

export default defineComponent({
  name: "MainLayout",

  components: {},
});
</script>

<script setup>
// user info
const userStore = useUserStore();

const avatar = "/web/avatar/default.webp";

async function logout() {
  api
    .post("/api/user/v1/logout")
    .then(() => {
      userStore.$reset();
      toRouter.Reload();
    })
    .catch(() => {});
}

onBeforeMount(async () => {
  // 初始化使用者登入狀態
  if (!userStore.dataLoaded) {
    await userStore.updateData();
  }
});
</script>
