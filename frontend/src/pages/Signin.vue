<template>
  <q-page class="signin-page">
    <q-card class="signin-form">
      <div>
        <div class="column q-pa-sm">
          <q-input
            outlined
            type="email"
            v-model="account"
            label="帳號(Email)"
            bottom-slots
            dense
            @update:model-value="inputUpdate"
          ></q-input>
        </div>
        <div class="column q-px-sm">
          <q-input
            outlined
            v-model="password"
            label="密碼"
            :type="showPwd ? 'password' : 'text'"
            bottom-slots
            dense
            @update:model-value="inputUpdate"
            ><template v-slot:append>
              <q-icon
                :name="showPwd ? 'visibility_off' : 'visibility'"
                class="cursor-pointer"
                @click="showPwd = !showPwd"
              /> </template
          ></q-input>
        </div>
        <div class="row q-px-sm items-center" v-if="loginFailed">
          <q-icon name="warning" class="text-red" style="font-size: 15px" />
          <div class="text-red-14">&ensp;{{ loginFailedMsg }}&ensp;</div>
        </div>

        <div class="row justify-center q-px-sm q-py-none">
          <q-btn
            class="q-mt-lg"
            color="primary"
            label="登入"
            :disable="!signinEnable"
            @click="sendSigninData"
          ></q-btn>
        </div>
      </div>
    </q-card>
  </q-page>
</template>

<style scoped>
.signin-page {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 100vh;
  background-color: #f0f0f0;
}

.signin-form {
  max-width: 500px;
  width: 100%;
  padding-top: 20px;
  padding-left: 10px;
  padding-right: 10px;
  padding-bottom: 20px;
  background-color: #fff;
  border-radius: 10px;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.2);
}

@media (max-width: 767px) {
  .signin-form {
    max-width: none;
  }
}
</style>

<script>
import { defineComponent, ref } from "vue";
import { api } from "boot/axios";
import { toRouter } from "boot/to-router";
import { useUserStore } from "stores/user";

export default defineComponent({
  name: "SigninPage",
  components: {},
});
</script>

<script setup>
const userStore = useUserStore();

const account = ref("");
const emailRule =
  /^\w+((-\w+)|(\.\w+))*\@[A-Za-z0-9]+((\.|-)[A-Za-z0-9]+)*\.[A-Za-z]+$/;
const password = ref("");
const showPwd = ref(true);
const signinEnable = ref(false);

function inputUpdate() {
  loginFailed.value = false;
  formatVaild();
}
function formatVaild() {
  if (account.value.search(emailRule) == -1) {
    signinEnable.value = false;
    return false;
  }
  if (password.value.length == 0) {
    signinEnable.value = false;
    return false;
  }

  signinEnable.value = true;
  return true;
}

const loginFailed = ref(false);
const loginFailedMsg = ref("");
async function sendSigninData() {
  signinEnable.value = false;
  loginFailed.value = false;
  await api
    .post("/api/user/v1/signin", {
      email: account.value,
      pwd: password.value,
    })
    .then(() => {
      // 成功登入 返回上一頁
      userStore.$reset();
      toRouter.PreviousPage(["/register"]);
    })
    .catch((error) => {
      if (error.response) {
        switch (error.response.status) {
          case 401:
            loginFailed.value = true;
            loginFailedMsg.value = "帳號或密碼錯誤";
            break;
          default:
            console.log("other error", error.response);
        }
      }
    });
  signinEnable.value = true;
}

const catchEnterKey = (evt) => {
  if (evt.which === 13 || evt.keyCode === 13) {
    if (signinEnable.value) {
      sendSigninData();
    }
  }
};
window.addEventListener("keyup", catchEnterKey);
</script>
