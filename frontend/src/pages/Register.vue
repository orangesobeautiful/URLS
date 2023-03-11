<template>
  <q-page class="flex flex-center">
    <q-dialog v-model="registerError">
      <q-card>
        <q-card-section>
          <div class="text-h6 text-center">註冊失敗</div>
        </q-card-section>

        <q-card-section class="q-pt-none">
          <p v-for="msg in registerErrMsgArray" :key="msg">
            {{ msg }}
          </p>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="確認" color="primary" v-close-popup />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <q-dialog v-model="registerSuccess">
      <q-card>
        <q-card-section>
          <div class="text-h6 text-center">註冊成功</div>
        </q-card-section>

        <q-card-section class="q-pt-none">
          {{ registerSuccessMsg }}
        </q-card-section>

        <q-card-actions align="right">
          <q-btn
            flat
            label="返回首頁"
            color="primary"
            v-close-popup
            @click="toRouter.HomePage()"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>

    <q-card class="register-container">
      <q-card-section class="bg-teal text-white">
        <div class="text-h6 text-center">註冊帳號</div>
        <div class="text-subtitle2"></div>
      </q-card-section>

      <q-card-actions vertical align="center">
        <div class="column input-container">
          <div class="column q-pa-sm">
            <q-input
              outlined
              dense
              bg-color="grey-4"
              type="email"
              v-model="email"
              label="電子郵件"
              bottom-slots
              hint=""
              lazy-rules
              :rules="[
                (val) => val.length <= 256 || '不支援長度超過256字元的電子郵件',
                (val) => val.search(emailRule) != -1 || '電子郵件格式錯誤',
              ]"
              @update:model-value="formatVaild"
            />
          </div>
          <div class="column q-pa-sm">
            <q-input
              ref="passwordInputRef"
              outlined
              dense
              bg-color="grey-4"
              v-model="password"
              type="password"
              label="密碼"
              bottom-slots
              :hint="'長度最少' + pwdMinLen + '個字元'"
              lazy-rules
              :rules="[
                (val) =>
                  pwdMinLen <= val.length ||
                  '密碼長度最少' + pwdMinLen + '個字元',
                (val) => val == rePassword || '兩次輸入的密碼不同',
              ]"
              @update:model-value="
                formatVaild();
                passwordVaild();
              "
            />
          </div>
          <div class="column q-pa-sm">
            <q-input
              ref="rePasswordInputRef"
              outlined
              dense
              bg-color="grey-4"
              v-model="rePassword"
              type="password"
              label="確認密碼"
              :rules="[
                (val) =>
                  pwdMinLen <= val.length ||
                  '密碼長度最少' + pwdMinLen + '個字元',
                (val) => val == password || '兩次輸入的密碼不同',
              ]"
              @update:model-value="
                formatVaild();
                passwordVaild();
              "
            />
          </div>
        </div>
        <div class="row justify-between q-px-md q-py-sm">
          <q-btn
            color="white"
            text-color="black"
            label="登入頁面"
            @click="toRouter.SigninPage()"
          />
          <q-btn
            color="deep-orange"
            glossy
            label="註冊"
            :disable="!registerEnable"
            @click="sendRegisterData"
          />
        </div>
      </q-card-actions>
    </q-card>
  </q-page>
</template>

<style lang="scss" scoped>
@import "src/css/width.scss";

.register-container {
  @include xs-width {
    width: 95%;
  }

  @media (min-width: $breakpoint-xs) {
    width: 538px;
  }
}

.input-container {
  @include xs-width {
    width: 100%;
  }

  @media (min-width: $breakpoint-xs) {
    width: 100%;
  }
}
.input {
  @include xs-width {
    width: 100%;
  }

  @media (min-width: $breakpoint-xs) {
    width: 100%;
  }
}
</style>

<script>
import { defineComponent, ref } from "vue";
import { api } from "boot/axios";

import { toRouter } from "boot/to-router";

export default defineComponent({
  name: "RegisterPage",
  components: {},
});
</script>

<script setup>
// const toRouter = new ToRouter();
const passwordInputRef = ref(null);
const rePasswordInputRef = ref(null);

const email = ref("");
const emailRule =
  /^\w+((-\w+)|(\.\w+))*\@[A-Za-z0-9]+((\.|-)[A-Za-z0-9]+)*\.[A-Za-z]+$/;
const pwdMinLen = 8;
const password = ref("");
const rePassword = ref("");
const passwordEqual = () => {
  if (password.value == rePassword.value) {
    return true;
  } else {
    return false;
  }
};
function passwordVaild() {
  if (passwordEqual() && password.value.length >= pwdMinLen) {
    passwordInputRef.value.resetValidation();
    passwordInputRef.value.resetValidation();
  }
}

const registerEnable = ref(false);

function formatVaild() {
  if (email.value.length > 256 || email.value.search(emailRule.value) == -1) {
    registerEnable.value = false;
    return false;
  }
  if (!passwordEqual() || password.value.length < pwdMinLen) {
    registerEnable.value = false;
    return false;
  }

  registerEnable.value = true;
  return true;
}

const registerSuccess = ref(false);
const registerSuccessMsg = ref("");
const registerError = ref(false);
const registerErrMsgArray = ref([]);
async function sendRegisterData() {
  await api
    .post("/api/user/v1/register", {
      email: email.value,
      pwd: password.value,
    })
    .then(() => {
      registerSuccess.value = true;
      userStore.$reset();
    })
    .catch((error) => {
      if (error.response) {
        registerError.value = true;
        registerErrMsgArray.value = [error.response.data["message"]];
      }
    });
}
</script>
