<template>
  <div id="app">
    <img alt="Vue logo" src="./assets/logo.png" />
    <div class="w-50">
      <b-input-group prepend="URL" class="mt-3">
        <b-form-input type="text" name="webUrl" v-model="webUrl"></b-form-input>
      </b-input-group>
      <div class="mt-2 w-100 d-flex justify-content-end">
        <b-button variant="info" class="mr-2" v-on:click="handleGetHistory()"
          >History</b-button
        >
        <b-button variant="primary" v-on:click="handleSearch()"
          >Search</b-button
        >
      </div>
      <div v-if="loading">
        <b-spinner label="Loading..." class="mt-2"></b-spinner>
      </div>
      <div
        v-if="mode === 'searchResult' && !loading"
        style="word-break: break-all"
        class="mt-3 mb-5"
      >
        <b-card
          style="text-align: left"
          header="Primary"
          header-bg-variant="primary"
          header-text-variant="white"
        >
          <template v-slot:header>
            <div class="w-100 d-flex justify-content-between">
              <div>
                <strong>{{ searchResult.webUrl }}</strong>
              </div>
              <div>
                {{ searchResult.is_down ? "Server down" : "Server active" }}
              </div>
            </div>
          </template>
          <div><strong>Title: </strong>{{ searchResult.title }}</div>
          <div><strong>Logo path: </strong>{{ searchResult.logo }}</div>
          <hr role="separator" aria-orientation="horizontal" />
          <div><strong>SSL grade: </strong>{{ searchResult.ssl_grade }}</div>
          <div>
            <strong>Previous SSL grade: </strong
            >{{ searchResult.previous_ssl_grade }}
          </div>
          <div>
            <strong>Servers have changed: </strong
            >{{ searchResult.servers_changed ? "Yes" : "No" }}
          </div>

          <div class="mt-2 mb-1"><strong>Servers:</strong></div>
          <b-card
            class="mt-2"
            v-for="server in searchResult.servers"
            :key="server.address"
            style="text-align: left"
            border-variant="primary"
          >
            <div class="w-100 d-flex justify-content-between">
              <div class="w-50">
                <strong>IP address: </strong>{{ server.address }}
              </div>
              <div class="w-50">
                <strong>Country: </strong>{{ server.country }}
              </div>
            </div>
            <div class="w-100 d-flex justify-content-between">
              <div class="w-50">
                <strong>SSL grade: </strong>{{ server.ssl_grade }}
              </div>
              <div class="w-50">
                <strong>Owner: </strong>{{ server.owner }}
              </div>
            </div>
          </b-card>
        </b-card>
      </div>
      <div
        v-if="mode === 'getHistory' && !loading"
        style="word-break: break-all"
        class="mt-3 mb-5"
      >
        <b-list-group>
          <b-list-group-item variant="info">
            <strong>Search History</strong>
          </b-list-group-item>
          <b-list-group-item
            v-for="item in getHistory.items"
            :key="item"
            style="text-align: left"
          >
            {{ item }}
          </b-list-group-item>
        </b-list-group>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "App",
  components: {},
  data: function() {
    return {
      webUrl: "",
      searchResult: null,
      getHistory: null,
      mode: "",
      loading: false
    };
  },
  methods: {
    handleSearch: async function() {
      if (!this.webUrl) {
        return;
      }
      this.loading = true;
      const response = await fetch(
        `http://localhost:8090/WebSearch?webURL=${this.webUrl}`
      );
      const json = await response.json();
      this.searchResult = { ...json, webUrl: this.webUrl };
      this.mode = "searchResult";
      this.loading = false;
    },
    handleGetHistory: async function() {
      this.loading = true;
      const response = await fetch(`http://localhost:8090/visited`);
      const json = await response.json();
      this.getHistory = json;
      this.mode = "getHistory";
      this.loading = false;
    }
  }
};
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  margin-top: 60px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}
* {
  box-sizing: content-box;
}
</style>
