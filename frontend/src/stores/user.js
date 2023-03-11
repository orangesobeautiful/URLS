import { defineStore } from "pinia";

import { api } from "boot/axios";

export const useUserStore = defineStore("user", {
  state: () => ({
    id: "",
    showName: "",
    role: 0,
    isManager: false,
    /** @type {string[]} */
    tags: [],
    normalQuota: 0,
    normalUsage: 0,
    customQuota: 0,
    customUsage: 0,
    hasLogin: false,
    dataLoaded: false,
  }),
  actions: {
    async updateData() {
      await api
        .get("/api/user/v1/self")
        .then((res) => {
          if (res) {
            const data = res.data;
            const userInfo = data["userInfo"];

            this.id = userInfo["idHex"];
            this.showName = userInfo["email"].split("@")[0];
            this.role = userInfo["role"];
            this.isManager = userInfo["isManager"];
            this.normalQuota = parseInt(userInfo["normalQuota"]);
            this.normalUsage = parseInt(userInfo["normalUsage"]);
            this.customQuota = parseInt(userInfo["customQuota"]);
            this.customUsage = parseInt(userInfo["customUsage"]);
            this.hasLogin = true;

            this.updateTags();
          }
        })
        .catch((error) => {
          if (error.response) {
            if (error.response.status == 401) {
              this.hasLogin = false;
            }
          }
        });
      this.dataLoaded = true;
    },

    async updateTags() {
      await api.get("/api/link/v1/tags").then((res) => {
        if (res) {
          const data = res.data;
          this.tags = data.tags;
        }
      });
    },
  },
});
