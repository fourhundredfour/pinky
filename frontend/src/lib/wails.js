// Thin wrapper around @wailsio/runtime so the rest of the frontend never
// spells out fully-qualified Go service/method names. Methods are invoked
// with Call.ByName instead of generated bindings, since bindings require
// running `wails3 generate bindings` as a build step this project does not
// depend on.
import { Call, Events } from '@wailsio/runtime';

const Service = {
  config: 'config.Service',
  tasks: 'tasks.Service',
  indicators: 'indicators.Service',
};

export function onConfigUpdate(callback) {
  return Events.On('config:update', callback);
}

export function onTasksUpdate(callback) {
  return Events.On('tasks:update', callback);
}

export function onClockTick(callback) {
  return Events.On('clock:tick', callback);
}

export function onIndicatorsUpdate(callback) {
  return Events.On('indicators:update', callback);
}

export function listTasks() {
  return Call.ByName(`${Service.tasks}.List`);
}

export function focusTask(id) {
  return Call.ByName(`${Service.tasks}.Focus`, id);
}

export function minimizeTask(id) {
  return Call.ByName(`${Service.tasks}.Minimize`, id);
}

export function closeTask(id) {
  return Call.ByName(`${Service.tasks}.Close`, id);
}

export function setVolume(level) {
  return Call.ByName(`${Service.indicators}.SetVolume`, level);
}

export function toggleMute() {
  return Call.ByName(`${Service.indicators}.ToggleMute`);
}

export function getIndicators() {
  return Call.ByName(`${Service.indicators}.Snapshot`);
}

export function openNetworkFlyout() {
  return Call.ByName(`${Service.indicators}.OpenNetworkFlyout`);
}

export function openSoundFlyout() {
  return Call.ByName(`${Service.indicators}.OpenSoundFlyout`);
}

export function openActionCenter() {
  return Call.ByName(`${Service.indicators}.OpenActionCenter`);
}

export function getConfig() {
  return Call.ByName(`${Service.config}.Get`);
}

export function saveConfig(cfg) {
  return Call.ByName(`${Service.config}.Save`, cfg);
}
