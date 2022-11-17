<script lang="ts" setup>
import {reactive} from 'vue'
import {LoadPaths} from '../wailsjs/go/main/App'
import {Validate} from '../wailsjs/go/main/App'

const data = reactive({
  spec: "",
  payload: "",
  paths: [""],
  path: "",
  errors: ""
})

function loadPaths(){
  data.paths.splice(0)
  LoadPaths(data.spec)
      .then(result => {
        data.paths.push(...result)
      })
}

function validateRequest(){
  data.errors = ""
  Validate(data.spec, data.payload, data.path)
      .then(result => {
        for (let resultKey in result) {
          data.errors += result[resultKey] + "\n"
        }
      })
}
</script>

<template>
  <div id="text-container">
    <textarea id="api" name="api" class="input-box" v-model="data.spec" title="API Spec"/>
    <textarea id="request" name="request" class="input-box" v-model="data.payload" title="Payload"/>
    <button class="btn" @click="loadPaths">Load Paths</button>
    <select id="paths" v-model="data.path" >
      <option v-for="s in data.paths" :value="s">
        {{ s }}
      </option>
    </select>
    <button class="btn" @click="validateRequest">Validate Payload</button>
    <div id="log-container">
      <textarea id="errors" class="error-box" v-model="data.errors" title="Errors" readonly/>
    </div>
  </div>
</template>

<style>

#text-container{
 height: 70%;
}
#log-container{
  height: 30%;
}
.input-box{
  display: inline-block;
  width: 48%;
  height: 95%;
  margin: 10px 10px 10px 10px;
  padding: 10px 10px 10px 10px;
  box-sizing: border-box;
  -webkit-box-sizing: border-box;
  -moz-box-sizing: border-box;
  resize: none;
}
.error-box{
  display: inline-block;
  width: 95%;
  height: 95%;
  margin: 10px 10px 10px 10px;
  padding: 10px 10px 10px 10px;
  box-sizing: border-box;
  -webkit-box-sizing: border-box;
  -moz-box-sizing: border-box;
  resize: none;
}
</style>
