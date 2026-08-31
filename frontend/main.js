const App = window.go.main.App;

let currentMenuId = null;
let activeId = -1;

function runtimeText(ms) {
  let total = Math.floor(ms / 1000);
  let d = Math.floor(total / 86400);
  let h = Math.floor((total % 86400) / 3600);
  let m = Math.floor((total % 3600) / 60);
  let parts = [];
  if (d > 0) parts.push(d + "d");
  if (h > 0) parts.push(h + "h");
  if (m > 0) parts.push(m + "m");
  if (parts.length === 0) parts.push("1m");
  return parts.join(" ");
}

function showError(err) {
  alert(String(err));
}

async function refresh() {
  let items;
  try {
    items = await App.GetItems();
  } catch (err) {
    showError(err);
    return;
  }
  const list = document.getElementById("file-list");
  list.innerHTML = "";
  document.getElementById("empty").classList.toggle("hidden", items.length > 0);

  items.forEach((item, id) => {
    const row = document.createElement("div");
    row.className = "row";
    row.dataset.id = id;

    const img = document.createElement("img");
    img.src = item.iconURL || "";
    img.alt = "";
    row.appendChild(img);

    const name = document.createElement("span");
    name.className = "name";
    name.textContent = item.title;
    name.title = item.title;
    row.appendChild(name);

    const runtime = document.createElement("span");
    runtime.className = "runtime";
    runtime.textContent = runtimeText(item.runtime_ms);
    row.appendChild(runtime);

    const runBtn = document.createElement("button");
    runBtn.className = "run";
    runBtn.textContent = item.running ? "Stop" : "Run";
    if (item.running) runBtn.classList.add("stop");
    runBtn.addEventListener("click", (e) => {
      e.stopPropagation();
      const call = item.running ? App.Stop(id) : App.Launch(id);
      call.catch(showError);
    });
    row.appendChild(runBtn);

    const menuBtn = document.createElement("button");
    menuBtn.className = "menu";
    menuBtn.textContent = "\u22EF";
    menuBtn.addEventListener("click", (e) => {
      e.stopPropagation();
      toggleMenu(id, menuBtn);
    });
    row.appendChild(menuBtn);

    row.addEventListener("dblclick", () => {
      if (item.running) return;
      App.Launch(id).catch(showError);
    });

    list.appendChild(row);
  });
}

function toggleMenu(id, anchor) {
  const menu = document.getElementById("menu");
  if (currentMenuId === id && !menu.classList.contains("hidden")) {
    menu.classList.add("hidden");
    currentMenuId = null;
    return;
  }
  currentMenuId = id;
  const r = anchor.getBoundingClientRect();
  menu.style.left = (r.left + window.scrollX) + "px";
  menu.style.top = (r.bottom + window.scrollY) + "px";
  menu.classList.remove("hidden");
}

document.getElementById("menu").addEventListener("click", async (e) => {
  const action = e.target.dataset.action;
  if (!action || currentMenuId === null) return;
  const id = currentMenuId;
  document.getElementById("menu").classList.add("hidden");
  currentMenuId = null;
  switch (action) {
    case "reveal":
      App.Reveal(id).catch(showError);
      break;
    case "rename":
      openRename(id);
      break;
    case "changeicon":
      App.ChangeIcon(id).catch(showError);
      break;
    case "updateicon":
      App.UpdateIcon(id).catch(showError);
      break;
    case "details":
      openDetails(id);
      break;
    case "delete":
      App.RemoveItem(id).catch(showError);
      break;
  }
});

document.addEventListener("click", (e) => {
  if (e.target.closest("#menu") || e.target.closest(".menu")) return;
  document.getElementById("menu").classList.add("hidden");
  currentMenuId = null;
});

function openRename(id) {
  activeId = id;
  document.getElementById("modal-title").textContent = "Rename";
  const input = document.getElementById("rename-input");
  input.classList.remove("hidden");
  input.value = "";
  document.getElementById("details-text").classList.add("hidden");
  document.getElementById("copy-btn").classList.add("hidden");
  document.getElementById("modal-ok").classList.remove("hidden");
  document.getElementById("modal-cancel").classList.remove("hidden");
  document.getElementById("modal").classList.remove("hidden");
  input.focus();
}

function openDetails(id) {
  activeId = id;
  document.getElementById("modal-title").textContent = "Details";
  document.getElementById("rename-input").classList.add("hidden");
  document.getElementById("modal-ok").classList.add("hidden");
  document.getElementById("modal-cancel").classList.remove("hidden");
  document.getElementById("copy-btn").classList.remove("hidden");
  const text = document.getElementById("details-text");
  text.classList.remove("hidden");
  text.value = "";
  App.Details(id)
    .then((json) => {
      text.value = json;
    })
    .catch(showError);
  document.getElementById("modal").classList.remove("hidden");
}

function closeModal() {
  document.getElementById("modal").classList.add("hidden");
}

document.getElementById("modal-ok").addEventListener("click", () => {
  const input = document.getElementById("rename-input");
  App.RenameItem(activeId, input.value).catch(showError);
  closeModal();
});

document.getElementById("modal-cancel").addEventListener("click", closeModal);

document.getElementById("copy-btn").addEventListener("click", () => {
  window.runtime.ClipboardSetText(document.getElementById("details-text").value);
});

document.getElementById("addBtn").addEventListener("click", () => {
  App.AddFiles().catch(showError);
});

window.runtime.OnFileDrop((_x, _y, paths) => {
  App.AddPaths(paths).catch(showError);
});

window.runtime.EventsOn("items:updated", refresh);

refresh();
