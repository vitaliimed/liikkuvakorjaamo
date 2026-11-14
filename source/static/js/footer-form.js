/* Liikkuva: a second, fully usable request form at the bottom of each landing page.
 * The server keeps /lead as the only intake route. A different form ID and field
 * IDs prevent accidental collisions with the already working hero-form controller.
 */
(function () {
  "use strict";
  document.addEventListener("DOMContentLoaded", function () {
    const mount = document.getElementById("footerLeadFormMount");
    const hero = document.querySelector("#lead-form .request-card");
    if (!mount || !hero) return;

    const card = hero.cloneNode(true);
    const form = card.querySelector("form");
    if (!form) return;
    // Do not leak entered contact data from the hero form into DOM clones.
    form.reset();
    for (const input of form.querySelectorAll('input[type="file"]')) input.value = "";
    const names = ["name", "phone", "email", "car", "address", "message"];
    for (const key of names) {
      const field = form.elements.namedItem(key);
      if (field) field.value = "";
    }
    const idMap = Object.create(null);
    card.querySelectorAll("[id]").forEach(function (element) {
      const original = element.id;
      idMap[original] = "footer-" + original;
      element.id = idMap[original];
    });
    card.querySelectorAll("label[for]").forEach(function (label) {
      if (idMap[label.htmlFor]) label.htmlFor = idMap[label.htmlFor];
    });
    card.querySelectorAll("[aria-controls]").forEach(function (control) {
      const original = control.getAttribute("aria-controls");
      if (idMap[original]) control.setAttribute("aria-controls", idMap[original]);
    });
    const source = form.elements.namedItem("source");
    if (source) source.value = "liikkuvakorjaamo.fi/footer";
    const message = card.querySelector(".form-message");
    const submit = form.querySelector('button[type="submit"]');
    const submitLabel = submit ? submit.textContent : "";
    const photos = form.elements.namedItem("photos");
    const trigger = card.querySelector(".photo-dropzone");
    const preview = card.querySelector(".photo-preview-list");
    const topForm = document.getElementById("leadForm");
    let thumbnailURLs = [];

    function say(text, kind) {
      if (!message) return;
      message.textContent = text || "";
      message.className = "form-message" + (kind ? " " + kind : "");
    }
    function setBusy(busy) {
      if (!submit) return;
      submit.disabled = busy;
      submit.textContent = busy ? (form.dataset.submitLoadingText || "Lähetetään…") : submitLabel;
    }
    function copyLocationFromTop() {
      if (!topForm) return;
      for (const fieldName of ["address", "location_source", "latitude", "longitude",
        "maps_url", "address_hint"]) {
        const original = topForm.elements.namedItem(fieldName);
        const target = form.elements.namedItem(fieldName);
        if (original && target && original.value) target.value = original.value;
      }
      if (form.elements.namedItem("address")?.value) say(
        form.dataset.lang === "fi" ? "Sijainti lisätty." : "Location added.", "success");
    }
    const mapButton = card.querySelector(".inline-actions [id$='pickMapBtn']");
    if (mapButton) mapButton.addEventListener("click", function () {
      const mainButton = document.getElementById("pickMapBtn");
      const modal = document.getElementById("mapPickerModal");
      if (!mainButton || !modal) return;
      const watch = new MutationObserver(function () {
        if (modal.classList.contains("open")) return;
        watch.disconnect();
        copyLocationFromTop();
      });
      watch.observe(modal, {attributes:true, attributeFilter:["class"]});
      mainButton.click();
    });
    const geoButton = card.querySelector(".inline-actions [id$='useLocationBtn']");
    if (geoButton) geoButton.addEventListener("click", function () {
      if (!navigator.geolocation) {
        say(form.dataset.geoUnsupported || "Geolocation unavailable", "error");
        return;
      }
      geoButton.disabled = true;
      navigator.geolocation.getCurrentPosition(async function (position) {
        form.elements.namedItem("latitude").value = String(position.coords.latitude);
        form.elements.namedItem("longitude").value = String(position.coords.longitude);
        form.elements.namedItem("location_source").value = "browser_geolocation";
        form.elements.namedItem("maps_url").value = "https://www.google.com/maps?q=" +
          encodeURIComponent(position.coords.latitude + "," + position.coords.longitude);
        try {
          const url = new URL("https://nominatim.openstreetmap.org/reverse");
          url.searchParams.set("format", "jsonv2");
          url.searchParams.set("lat", String(position.coords.latitude));
          url.searchParams.set("lon", String(position.coords.longitude));
          const result = await fetch(url.toString()).then(function (r) {return r.json()});
          if (result.display_name) form.elements.namedItem("address").value = result.display_name;
        } catch (_) { /* Coordinates are still available; address can be entered manually. */ }
        say(form.dataset.geoAdded || "Location added", "success");
        geoButton.disabled = false;
      }, function () {
        say(form.dataset.geoFailed || "Could not determine location", "error");
        geoButton.disabled = false;
      }, {enableHighAccuracy:false,timeout:12000});
    });
    function showPhotos() {
      if (!preview || !photos) return;
      thumbnailURLs.forEach(URL.revokeObjectURL);
      thumbnailURLs = [];
      preview.replaceChildren();
      Array.from(photos.files || []).slice(0, 3).forEach(function (file) {
        if (!file.type.startsWith("image/")) return;
        const url = URL.createObjectURL(file);
        thumbnailURLs.push(url);
        const thumb = document.createElement("img");
        thumb.src = url;
        thumb.alt = file.name;
        thumb.style.cssText = "width:65px;height:65px;object-fit:cover;border-radius:8px;margin-right:6px";
        preview.append(thumb);
      });
    }
    if (trigger && photos) trigger.addEventListener("click", () => photos.click());
    if (photos) photos.addEventListener("change", showPhotos);

    const phone = form.elements.namedItem("phone");
    const plate = form.elements.namedItem("car");
    if (plate) plate.addEventListener("input", function () {
      const raw = this.value.toUpperCase().replace(/[^A-Z0-9]/g, "");
      const match = raw.match(/^([A-Z]{0,3})([0-9]{0,3})$/);
      if (match) this.value = match[1] + (match[2] ? "-" + match[2] : "");
    });
    form.addEventListener("submit", async function (event) {
      event.preventDefault();
      say("", "");
      if (!form.reportValidity()) return;
      if (photos && (photos.files.length > 3 || Array.from(photos.files).some(f =>
        !f.type.startsWith("image/") || f.size > 10 * 1024 * 1024))) {
        say(form.dataset.photosTooMany || "Please use up to three images under 10 MB.", "error");
        return;
      }
      if (plate && !/^[A-Z]{3}-[0-9]{1,3}$/.test(plate.value.toUpperCase())) {
        say(form.dataset.registrationInvalid || "Invalid registration number", "error");
        plate.focus();
        return;
      }
      if (phone && !phone.value.trim()) {
        say(form.dataset.contactRequired || "Phone number required", "error");
        phone.focus();
        return;
      }
      const fields = new FormData(form);
      fields.set("page_url", location.href);
      fields.set("referer", document.referrer);
      const utms = new URLSearchParams(location.search);
      ["source", "medium", "campaign", "term", "content"].forEach(function (name) {
        if (utms.has("utm_" + name)) fields.set("utm_" + name, utms.get("utm_" + name));
      });
      setBusy(true);
      try {
        const response = await fetch("/lead", {method:"POST", body:fields, credentials:"same-origin"});
        const result = await response.json();
        if (!response.ok || !result.ok) {
          const problem = result.errors ? Object.values(result.errors).join(", ") : result.message;
          throw new Error(problem || form.dataset.errorMessage || "Could not send the request");
        }
        if (typeof window.liikkuvaTrackCustomEvent === "function") {
          window.liikkuvaTrackCustomEvent("lead_submit_success", {
            site_brand:"liikkuva", form_location:"footer", page_language:form.dataset.lang || "fi"
          });
        }
        if (result.redirect_url) {
          window.location.assign(result.redirect_url);
          return;
        }
        say(result.message || form.dataset.successMessage, "success");
        form.reset();
        showPhotos();
      } catch (error) {
        say(error.message || form.dataset.networkError || "Network error", "error");
      } finally {
        setBusy(false);
      }
    });
    mount.replaceChildren(card);
    mount.classList.add("is-ready");
  });
})();
