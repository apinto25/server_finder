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
      <div v-if="mode === 'searchResult'" style="word-break: break-all">
        {{ JSON.stringify(searchResult) }}
      </div>
      <div v-if="mode === 'getHistory'" style="word-break: break-all">
        {{ JSON.stringify(getHistory) }}
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
      mode: ""
    };
  },
  methods: {
    handleSearch: async function() {
      if (!this.webUrl) {
        return;
      }
      const response = await fetch(
        `http://192.168.0.19:8090/WebSearch?webURL=${this.webUrl}`
      );
      const json = await response.json();
      this.searchResult = json;
      this.mode = "searchResult";
    },
    handleGetHistory: async function() {
      const response = await fetch(`http://192.168.0.19:8090/visited`);
      const json = await response.json();
      this.getHistory = json;
      this.mode = "getHistory";
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
