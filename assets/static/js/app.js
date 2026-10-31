document.addEventListener("DOMContentLoaded", function () {
 document.querySelectorAll('[data-lead-form]').forEach(initLeadForm);
});
function initLeadForm(form) {
  const prefix = form.dataset.idPrefix || '';
  const byId = id => document.getElementById(prefix + id);

  const messageBox = byId("formMessage");
  const submitButton = form.querySelector("button[type='submit']");
  const pageUrlInput = byId("pageUrl");
  const refererField = byId("refererField");
  const utmSource = byId("utmSource");
  const utmMedium = byId("utmMedium");
  const utmCampaign = byId("utmCampaign");
  const utmTerm = byId("utmTerm");
  const utmContent = byId("utmContent");

  const nameInput = byId("name");
  const phoneInput = byId("phone");
  const emailInput = byId("email");
  const addressInput = byId("address");
  const addressHintInput = byId("addressHint");
  const carInput = byId("car");
  const messageInput = byId("message");
  const photosInput = byId("photos");
  const photosTrigger = byId("photosTrigger");
  const photoPreviewList = byId("photoPreviewList");
  const mobileStickyFormLink = prefix ? null : document.getElementById("mobileStickyFormLink");

  const useLocationBtn = byId("useLocationBtn");
  const pickMapBtn = byId("pickMapBtn");
  const locationSourceInput = byId("locationSource");
  const latitudeInput = byId("latitude");
  const longitudeInput = byId("longitude");
  const mapsUrlInput = byId("mapsUrl");

  const mapModal = byId("mapPickerModal");
  const closeMapModalBtn = byId("closeMapModal");
  const confirmMapPointBtn = byId("confirmMapPoint");
  const mapCanvas = byId("mapPickerCanvas");

  const areasSection = prefix ? null : document.getElementById("areas");
  const areasShell = areasSection ? areasSection.querySelector(".areas-shell") : null;
  const areasLiveMapEl = prefix ? null : document.getElementById("areasLiveMap");

  const i18n = {
    successMessage: form.dataset.successMessage || "",
    errorMessage: form.dataset.errorMessage || "",
    submitLoadingText: form.dataset.submitLoadingText || "Sending...",
    nameRequired: form.dataset.nameRequired || "Please enter your name.",
    contactRequired: form.dataset.contactRequired || "Please enter your phone number.",
    phoneInvalid: form.dataset.phoneInvalid || "Please enter a valid phone number.",
    emailInvalid: form.dataset.emailInvalid || "Please enter a valid email address.",
    registrationInvalid: form.dataset.registrationInvalid || "Please enter a valid registration number.",
    photosTooMany: form.dataset.photosTooMany || "Please upload up to 3 photos only.",
    photosType: form.dataset.photosType || "Only image files are allowed.",
    photosSize: form.dataset.photosSize || "Each photo must be smaller than 10 MB.",
    geoUnsupported: form.dataset.geoUnsupported || "Geolocation is not supported in this browser.",
    geoLoading: form.dataset.geoLoading || "Detecting location...",
    geoAdded: form.dataset.geoAdded || "Location added",
    geoFailed: form.dataset.geoFailed || "Could not get your location.",
    mapPickFirst: form.dataset.mapPickFirst || "Please pick a point on the map first.",
    mapPointAdded: form.dataset.mapPointAdded || "Map point added",
    reverseGeocodeFailed: form.dataset.reverseGeocodeFailed || "Coordinates were saved, but the address could not be filled automatically.",
    networkError: form.dataset.networkError || "Network or browser error. Please try again.",
    lang: form.dataset.lang || "en"
  };

  const cookiePrefix = "liikkuva_";
  const leadSuccessStorageKey = "liikkuva_lead_success_v1";
  const fieldCookies = []; // Do not persist personal form fields in browser cookies.

  const mapAssets = {
    css: mapModal ? mapModal.dataset.leafletCss : "",
    js: mapModal ? mapModal.dataset.leafletJs : ""
  };



  let hasTrackedFormStart = false;
  let originalSubmitText = submitButton ? submitButton.textContent.trim() : "";
  let selectedLat = "";
  let selectedLng = "";
  let pickerMap = null;
  let pickerMarker = null;
  let leafletReadyPromise = null;
  let areasLiveMap = null;
  let areasResizeTimer = null;
  let fittingAreasMap = false;
  let selectedFiles = [];
  let locationLookupSerial = 0;

  function trackEvent(name, params) {
    if (typeof window.liikkuvaTrackEvent === 'function') return window.liikkuvaTrackEvent(name, Object.assign({form_position: form.dataset.formPosition}, params || {}));
    return false;
  }

  function trackFormStart() {
    const c=typeof window.liikkuvaConsentState === "function" ? window.liikkuvaConsentState() : {};
    if(!c.analytics) return;
    if (hasTrackedFormStart) return;
    hasTrackedFormStart = trackEvent("lead_form_start", {
      page_language: i18n.lang
    });
  }

  function clearMessage() {
    if (!messageBox) return;
    messageBox.textContent = "";
    messageBox.className = "form-message";
  }

  function showMessage(text, type) {
    if (!messageBox) return;
    messageBox.textContent = text || "";
    messageBox.className = "form-message";
    if (type) messageBox.classList.add(type);
  }

  function setSubmitState(disabled) {
    if (!submitButton) return;
    submitButton.disabled = !!disabled;
    submitButton.classList.toggle("is-loading", !!disabled);
    submitButton.setAttribute("aria-busy", disabled ? "true" : "false");
    submitButton.textContent = disabled ? i18n.submitLoadingText : originalSubmitText;
  }

  function setButtonState(button, text, disabled) {
    if (!button) return;
    button.textContent = text;
    button.disabled = !!disabled;
  }

  function setCookie(name, value, days) { /* Personal form data is not stored in cookies. */
  }

  function getCookie(name) {
    const needle = `${cookiePrefix}${name}=`;
    const parts = document.cookie.split(";");
    for (const part of parts) {
      const trimmed = part.trim();
      if (trimmed.startsWith(needle)) {
        return decodeURIComponent(trimmed.slice(needle.length));
      }
    }
    return "";
  }

  function restoreFieldCookies() {
    for (const [name, input] of fieldCookies) {
      if (!input) continue;
      const value = getCookie(name);
      if (!value) continue;
      input.value = value;
    }

    if (phoneInput && phoneInput.value) {
      phoneInput.value = formatFinnishPhone(phoneInput.value);
    }

    if (carInput && carInput.value) {
      carInput.value = normalizeRegistrationNumber(carInput.value);
    }

    if (messageInput) {
      autoResizeTextarea(messageInput);
    }
  }

  function bindFieldCookies() {
    for (const [name, input] of fieldCookies) {
      if (!input) continue;
      input.addEventListener("input", function () {
        setCookie(name, input.value, 30);
      });
      input.addEventListener("change", function () {
        setCookie(name, input.value, 30);
      });
    }
  }

  function fillTrackingFields() {
    const params = new URLSearchParams(window.location.search);

    if (pageUrlInput) pageUrlInput.value = window.location.href;
    if (refererField) refererField.value = document.referrer || "";

    if (utmSource) utmSource.value = params.get("utm_source") || "";
    if (utmMedium) utmMedium.value = params.get("utm_medium") || "";
    if (utmCampaign) utmCampaign.value = params.get("utm_campaign") || "";
    if (utmTerm) utmTerm.value = params.get("utm_term") || "";
    if (utmContent) utmContent.value = params.get("utm_content") || "";
  }

  function onlyDigits(value) {
    return String(value || "").replace(/\D/g, "");
  }

  function formatFinnishPhone(value) {
    let digits = onlyDigits(value);

    if (digits.startsWith("358")) {
      digits = digits.slice(3);
    }
    if (digits.startsWith("0")) {
      digits = digits.slice(1);
    }

    digits = digits.slice(0, 9);

    if (!digits) return "+358 ";

    const parts = [];
    if (digits.length > 0) parts.push(digits.slice(0, 2));
    if (digits.length > 2) parts.push(digits.slice(2, 5));
    if (digits.length > 5) parts.push(digits.slice(5, 7));
    if (digits.length > 7) parts.push(digits.slice(7, 9));

    return "+358 " + parts.join(" ").trim();
  }

  function isValidPhone(value) {
    const digits = onlyDigits(value);
    if (!digits) return false;
    if (digits.startsWith("358")) return digits.length === 12;
    if (digits.startsWith("0")) return digits.length === 10;
    return false;
  }

  function normalizeRegistrationNumber(value) {
    const raw = String(value || "").toUpperCase();
    let letters = "";
    let digits = "";
    let mode = "letters";

    for (const ch of raw) {
      if (mode === "letters") {
        if (/[A-Z]/.test(ch)) {
          if (letters.length < 3) letters += ch;
          continue;
        }

        if (/[0-9]/.test(ch)) {
          if (letters.length === 3) {
            mode = "digits";
            if (digits.length < 3) digits += ch;
          }
          continue;
        }

        if (ch === "-" && letters.length === 3) {
          mode = "digits";
        }
        continue;
      }

      if (/[0-9]/.test(ch) && digits.length < 3) {
        digits += ch;
      }
    }

    if (!letters) return "";
    if (letters.length < 3) return letters;
    if (digits.length === 0) return letters + "-";
    return letters + "-" + digits;
  }

  function isValidRegistrationNumber(value) {
    if (!value) return true;
    return /^[A-Z]{3}-[0-9]{3}$/.test(value);
  }

  function isValidEmail(value) {
    if (!value) return false;
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
  }

  function setLocationData(lat, lng, source) {
    selectedLat = String(lat);
    selectedLng = String(lng);

    if (latitudeInput) latitudeInput.value = selectedLat;
    if (longitudeInput) longitudeInput.value = selectedLng;
    if (locationSourceInput) locationSourceInput.value = source || "";
    if (mapsUrlInput) {
      mapsUrlInput.value = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(selectedLat + "," + selectedLng)}`;
    }
  }

  function resetLocationData() {
    selectedLat = "";
    selectedLng = "";

    if (latitudeInput) latitudeInput.value = "";
    if (longitudeInput) longitudeInput.value = "";
    if (locationSourceInput) locationSourceInput.value = "";
    if (mapsUrlInput) mapsUrlInput.value = "";
    if (addressHintInput) addressHintInput.value = "";
  }

  function validatePhotos() {
    if (!selectedFiles.length) return { ok: true, message: "" };

    if (selectedFiles.length > 3) {
      return { ok: false, message: i18n.photosTooMany };
    }

    for (const file of selectedFiles) {
      if (!file.type.startsWith("image/")) {
        return { ok: false, message: i18n.photosType };
      }
      if (file.size > 10 * 1024 * 1024) {
        return { ok: false, message: i18n.photosSize };
      }
    }

    return { ok: true, message: "" };
  }

  function syncSelectedFilesToInput() {
    if (!photosInput) return;
    const dt = new DataTransfer();
    selectedFiles.forEach((file) => dt.items.add(file));
    photosInput.files = dt.files;
  }

  function renderPhotoPreviews() {
    const count = form.querySelector("[data-photo-count]");
    if(count)count.textContent = selectedFiles.length + " / 3";
    if (!photoPreviewList) return;
    photoPreviewList.innerHTML = "";

    selectedFiles.forEach((file, index) => {
      const item = document.createElement("div");
      item.className = "photo-preview-item";

      const thumb = document.createElement("img");
      thumb.alt = file.name;
      thumb.src = URL.createObjectURL(file);
      thumb.onload = function () {
        URL.revokeObjectURL(thumb.src);
      };

      const remove = document.createElement("button");
      remove.type = "button";
      remove.className = "photo-remove-btn";
      remove.setAttribute("aria-label", "Remove photo");
      remove.textContent = "×";
      remove.addEventListener("click", function () {
        selectedFiles = selectedFiles.filter((_, i) => i !== index);
        syncSelectedFilesToInput();
        renderPhotoPreviews();
      });

      item.appendChild(thumb);
      item.appendChild(remove);
      photoPreviewList.appendChild(item);
    });
  }

  function handlePhotoSelection(fileList) {
    const incoming = Array.from(fileList || []);
    if (!incoming.length) return;

    selectedFiles = selectedFiles.concat(incoming).slice(0, 3);

    const validation = validatePhotos();
    if (!validation.ok) {
      selectedFiles = selectedFiles.slice(0, 3);
      trackEvent("lead_photo_upload", {
        page_language: i18n.lang,
        photo_count: selectedFiles.length
      });
      showMessage(validation.message, "error");
      syncSelectedFilesToInput();
      renderPhotoPreviews();
      return;
    }

    trackEvent("lead_photo_upload", {
      page_language: i18n.lang,
      photo_count: selectedFiles.length
    });

    syncSelectedFilesToInput();
    renderPhotoPreviews();
    clearMessage();
  }

  function autoResizeTextarea(textarea) {
    if (!textarea) return;
    textarea.style.height = "auto";
    textarea.style.height = Math.min(textarea.scrollHeight, window.matchMedia("(min-width:981px)").matches ? 80 : 180) + "px";
  }

  function ensureLeafletAssets() {
    if (window.L) return Promise.resolve();
    if (!mapAssets.js) return Promise.reject(new Error("Leaflet URL is missing"));
    if (leafletReadyPromise) return leafletReadyPromise;

    leafletReadyPromise = new Promise((resolve, reject) => {
      if (mapAssets.css && !document.querySelector(`link[href="${mapAssets.css}"]`)) {
        const css = document.createElement("link");
        css.rel = "stylesheet";
        css.href = mapAssets.css;
        document.head.appendChild(css);
      }

      if (document.querySelector(`script[src="${mapAssets.js}"]`)) {
        const poll = window.setInterval(function () {
          if (window.L) {
            window.clearInterval(poll);
            resolve();
          }
        }, 50);
        window.setTimeout(function () {
          window.clearInterval(poll);
          if (!window.L) reject(new Error("Leaflet failed to load"));
        }, 5000);
        return;
      }

      const script = document.createElement("script");
      script.src = mapAssets.js;
      script.async = true;
      script.onload = () => resolve();
      script.onerror = () => reject(new Error("Leaflet failed to load"));
      document.body.appendChild(script);
    });

    return leafletReadyPromise;
  }

  function fitAreasMap() {
    if (!areasLiveMap || !window.L) return;
    fittingAreasMap = true;
    try {
      areasLiveMap.invalidateSize();

    } finally { fittingAreasMap = false; }
  }

  async function initAreasLiveMap() {
    if (!areasSection || !areasShell || !areasLiveMapEl || areasLiveMap) return;

    try {
      await ensureLeafletAssets();
    } catch (error) {
      console.error("Areas map load failed", error);
      return;
    }

    areasLiveMapEl.hidden = false;

    areasLiveMap = window.L.map(areasLiveMapEl, {
      zoomControl: true,
      scrollWheelZoom: false,
      dragging: true,
      doubleClickZoom: true,
      touchZoom: true,
      boxZoom: false,
      keyboard: false,
      tap: true
    });

    window.L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
      attribution: "© OpenStreetMap contributors"
    }).addTo(areasLiveMap);

    areasLiveMap.setView([60.1699, 24.9384], 12, {animate: false});
    fitAreasMap();
    areasLiveMap.on('zoomstart', () => { if (!fittingAreasMap) trackEvent('service_area_map_interaction', {action: 'zoom'}); });
    areasLiveMap.on('dragstart', () => trackEvent('service_area_map_interaction', {action: 'move'}));
    areasShell.classList.add("is-live-map");

    window.addEventListener("resize", function () {
      if (!areasLiveMap) return;
      clearTimeout(areasResizeTimer);
      areasResizeTimer = setTimeout(fitAreasMap, 120);
    });
  }

  function watchAreasLiveMap() {
    if (!areasSection || !areasShell || !areasLiveMapEl) return;

    if (!("IntersectionObserver" in window)) {
      initAreasLiveMap();
      return;
    }

    const observer = new IntersectionObserver(function (entries) {
      const entry = entries[0];
      if (!entry || !entry.isIntersecting) return;
      initAreasLiveMap();
      observer.disconnect();
    }, {
      rootMargin: "240px 0px"
    });

    observer.observe(areasSection);
  }

  async function reverseGeocode(lat, lng) {
    const url = new URL("https://nominatim.openstreetmap.org/reverse");
    url.searchParams.set("format", "jsonv2");
    url.searchParams.set("lat", String(lat));
    url.searchParams.set("lon", String(lng));
    url.searchParams.set("accept-language", i18n.lang);

    const response = await fetch(url.toString(), {
      headers: { Accept: "application/json" }
    });

    if (!response.ok) throw new Error("reverse geocode failed");

    const data = await response.json();
    return data && data.display_name ? data.display_name : "";
  }

  async function resolveAddressFromCoordinates(lat, lng) {
    if (!addressInput) return;
    const serial = ++locationLookupSerial;
    const previousAddress = addressInput.value;
    const coordinateAddress = `${Number(lat).toFixed(5)}, ${Number(lng).toFixed(5)}`;
    const fallbackAddress = () => {
      if (serial === locationLookupSerial && addressInput.value === previousAddress) addressInput.value = coordinateAddress;
    };
    try {
      const address = await reverseGeocode(lat, lng);
      // A slower response must not overwrite a newer location or a manual correction.
      if (serial !== locationLookupSerial || addressInput.value !== previousAddress) return;
      if (address) {
        addressInput.value = address;
        setCookie("address", address, 30);
      } else {
        fallbackAddress();
        showMessage(i18n.reverseGeocodeFailed, "error");
      }
    } catch (error) {
      console.error(error);
      fallbackAddress();
      showMessage(i18n.reverseGeocodeFailed, "error");
    }
  }

  async function openMapModal() {
    if (!mapModal || !mapCanvas) return;
    clearMessage();

    try {
      await ensureLeafletAssets();
    } catch (error) {
      console.error(error);
      showMessage(i18n.networkError, "error");
      return;
    }

    document.body.classList.add("modal-open");
    mapModal.classList.add("open");
    closeMapModalBtn?.focus({preventScroll: true});

    setTimeout(function () {
      if (!pickerMap) {
        pickerMap = window.L.map(mapCanvas, {
          center: [60.1699, 24.9384],
          zoom: 12,
          zoomControl: true
        });

        window.L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
          maxZoom: 19,
          attribution: "Map data © OpenStreetMap contributors"
        }).addTo(pickerMap);

        pickerMap.on("click", function (e) {
          const lat = e.latlng.lat;
          const lng = e.latlng.lng;
          selectedLat = String(lat);
          selectedLng = String(lng);

          if (!pickerMarker) {
            pickerMarker = window.L.marker([lat, lng], { draggable: true }).addTo(pickerMap);
            pickerMarker.on("dragend", function () {
              const pos = pickerMarker.getLatLng();
              selectedLat = String(pos.lat);
              selectedLng = String(pos.lng);
            });
          } else {
            pickerMarker.setLatLng([lat, lng]);
          }
        });
      }

      pickerMap.invalidateSize();

      if (selectedLat && selectedLng) {
        const lat = Number(selectedLat);
        const lng = Number(selectedLng);
        pickerMap.setView([lat, lng], 15);
        if (!pickerMarker) {
          pickerMarker = window.L.marker([lat, lng], { draggable: true }).addTo(pickerMap);
        } else {
          pickerMarker.setLatLng([lat, lng]);
        }
      }
    }, 150);
  }

  function closeMapModal() {
    if (!mapModal) return;
    const wasOpen = mapModal.classList.contains("open");
    mapModal.classList.remove("open");
    document.body.classList.remove("modal-open");
    if(wasOpen)pickMapBtn?.focus({preventScroll: true});
  }

  function handleUseLocation() {
    clearMessage();

    if (!navigator.geolocation) {
      showMessage(i18n.geoUnsupported, "error");
      return;
    }

    setButtonState(useLocationBtn, i18n.geoLoading, true);

    navigator.geolocation.getCurrentPosition(
      async function (position) {
        const lat = position.coords.latitude;
        const lng = position.coords.longitude;

        setLocationData(lat, lng, "geolocation");
        await resolveAddressFromCoordinates(lat, lng);
        setButtonState(useLocationBtn, i18n.geoAdded, false);
      },
      function (error) {
        console.error(error);
        setButtonState(useLocationBtn, useLocationBtn.dataset.defaultLabel || "Use my location", false);
        showMessage(i18n.geoFailed, "error");
      },
      {
        enableHighAccuracy: true,
        timeout: 10000,
        maximumAge: 0
      }
    );
  }

  function rememberSuccessfulLead() {
    try {
      sessionStorage.setItem(leadSuccessStorageKey, JSON.stringify({
        savedAt: new Date().toISOString(),
        pageLanguage: i18n.lang,
        hasPhotos: selectedFiles.length > 0,
        hasLocation: !!(latitudeInput && latitudeInput.value && longitudeInput && longitudeInput.value)
      }));
    } catch (error) {
      console.error("lead success sessionStorage failed", error);
    }
  }

  async function handleSubmit(event) {
    event.preventDefault();

    trackEvent("lead_submit_attempt", {
      page_language: i18n.lang,
      has_phone: !!(phoneInput && phoneInput.value.trim()),
      has_email: !!(emailInput && emailInput.value.trim()),
      has_photos: selectedFiles.length > 0,
      has_location: !!(latitudeInput && latitudeInput.value && longitudeInput && longitudeInput.value)
    });

    clearMessage();
    setSubmitState(true);
    fillTrackingFields();

    if (phoneInput) {
      phoneInput.value = formatFinnishPhone(phoneInput.value);
      setCookie("phone", phoneInput.value, 30);
    }

    if (carInput) {
      carInput.value = normalizeRegistrationNumber(carInput.value);
      setCookie("car", carInput.value, 30);
    }

    const nameValue = nameInput ? nameInput.value.trim() : "";
    const phoneValue = phoneInput ? phoneInput.value.trim() : "";
    const emailValue = emailInput ? emailInput.value.trim() : "";
    const carValue = carInput ? carInput.value.trim() : "";

    if (!nameValue) {
      trackEvent("lead_validation_error", {field_name: "name"});
      showMessage(i18n.nameRequired, "error");
      setSubmitState(false);
      if (nameInput) nameInput.focus();
      return;
    }

    if (!phoneValue) {
      trackEvent("lead_validation_error", {field_name: "phone"});
      showMessage(i18n.contactRequired, "error");
      setSubmitState(false);
      if (phoneInput) phoneInput.focus();
      return;
    }

    if (phoneValue && !isValidPhone(phoneValue)) {
      trackEvent("lead_validation_error", {field_name: "phone"});
      showMessage(i18n.phoneInvalid, "error");
      setSubmitState(false);
      if (phoneInput) phoneInput.focus();
      return;
    }

    if (emailValue && !isValidEmail(emailValue)) {
      trackEvent("lead_validation_error", {field_name: "email"});
      showMessage(i18n.emailInvalid, "error");
      setSubmitState(false);
      if (emailInput) emailInput.focus();
      return;
    }

    if (!carValue || !isValidRegistrationNumber(carValue)) {
      trackEvent("lead_validation_error", {field_name: "car"});
      showMessage(i18n.registrationInvalid, "error");
      setSubmitState(false);
      if (carInput) carInput.focus();
      return;
    }

    if (!addressInput || !addressInput.value.trim()) {
      trackEvent("lead_validation_error", {field_name: "address"});
      showMessage(i18n.lang === "fi" ? "Anna auton osoite tai valitse sijainti kartalta." : "Enter the car address or choose its location on the map.", "error");
      setSubmitState(false);
      if (addressInput) addressInput.focus();
      return;
    }

    const photoValidation = validatePhotos();
    if (!photoValidation.ok) {
      showMessage(photoValidation.message, "error");
      setSubmitState(false);
      return;
    }

    try {
      syncSelectedFilesToInput();

      const formData = new FormData(form);
      const response = await fetch("/lead", {
        method: "POST",
        body: formData,
        credentials: "same-origin"
      });

      let result = null;
      try {
        result = await response.json();
      } catch (jsonError) {
        console.error("lead json parse failed", jsonError);
      }

      if (response.ok && result && result.ok) {
        rememberSuccessfulLead();
        if(window.liikkuvaSavedLead) window.liikkuvaSavedLead(result.submission_id, form.dataset.formPosition);
        

        trackEvent("lead_submit_success", {
          page_language: i18n.lang,
          has_photos: selectedFiles.length > 0,
          has_location: !!(latitudeInput && latitudeInput.value && longitudeInput && longitudeInput.value)
        });

        if (result.redirect_url) {
          setTimeout(function(){window.location.assign(result.redirect_url);}, 700);
          return;
        }

        showMessage(result.message || i18n.successMessage, "success");
        form.reset();
        selectedFiles = [];
        renderPhotoPreviews();
        fillTrackingFields();
        resetLocationData();
        restoreFieldCookies();
        return;
      }

      let finalErrorText = (result && result.message) || i18n.errorMessage;
      if (result && result.errors && typeof result.errors === "object") {
        const values = Object.values(result.errors).filter(Boolean);
        if (values.length > 0) {
          finalErrorText = values.join(", ");
        }
      }

      trackEvent("lead_submit_error", {
        page_language: i18n.lang,
        status_code: response.status || 0
      });
      showMessage(finalErrorText, "error");
    } catch (error) {
      console.error(error);
      trackEvent("lead_submit_network_error", {
        page_language: i18n.lang
      });
      showMessage(i18n.networkError, "error");
    } finally {
      setSubmitState(false);
    }
  }

  fillTrackingFields();
  restoreFieldCookies();
  bindFieldCookies();
  watchAreasLiveMap();

  function focusFirstUsefulField() {
    const candidates = [nameInput, phoneInput, messageInput];
    const firstEmpty = candidates.find(function (input) {
      return input && !String(input.value || "").trim();
    });
    const target = firstEmpty || nameInput || phoneInput || messageInput;
    if (!target) return;
    setTimeout(function () {
      try {
        target.focus({ preventScroll: true });
      } catch (error) {
        target.focus();
      }
    }, 420);
  }

  [nameInput, phoneInput, emailInput, addressInput, addressHintInput, carInput, messageInput].forEach(function (input) {
    if (!input) return;
    input.addEventListener("focus", trackFormStart);
    input.addEventListener("input", trackFormStart);
  });

  if (useLocationBtn) {
    useLocationBtn.dataset.defaultLabel = useLocationBtn.textContent;
    useLocationBtn.addEventListener("click", function () {
      trackEvent("lead_use_location_click", {
        page_language: i18n.lang
      });
      handleUseLocation();
    });
  }

  if (pickMapBtn) {
    pickMapBtn.dataset.defaultLabel = pickMapBtn.textContent;
    pickMapBtn.addEventListener("click", function () {
      trackEvent("lead_map_open", {
        page_language: i18n.lang
      });
      openMapModal();
    });
  }

  if (closeMapModalBtn) {
    closeMapModalBtn.addEventListener("click", closeMapModal);
  }

  if (confirmMapPointBtn) {
    confirmMapPointBtn.addEventListener("click", async function () {
      if (!selectedLat || !selectedLng) {
        showMessage(i18n.mapPickFirst, "error");
        return;
      }

      setLocationData(selectedLat, selectedLng, "map_picker");
      trackEvent("lead_map_point_confirmed", {
        page_language: i18n.lang
      });
      await resolveAddressFromCoordinates(selectedLat, selectedLng);
      closeMapModal();
      setButtonState(pickMapBtn, i18n.mapPointAdded, false);
    });
  }

  document.addEventListener("keydown", event => {if(event.key === "Escape" && mapModal?.classList.contains("open"))closeMapModal();});

  if (mapModal) {
    mapModal.addEventListener("click", function (event) {
      if (event.target === mapModal) closeMapModal();
    });
  }

  if (phoneInput) {
    phoneInput.addEventListener("focus", function () {
      if (!phoneInput.value) phoneInput.value = "+358 ";
    });

    phoneInput.addEventListener("input", function () {
      phoneInput.value = formatFinnishPhone(phoneInput.value);
      setCookie("phone", phoneInput.value, 30);
    });

    phoneInput.addEventListener("paste", function (event) {
      const pasted = event.clipboardData?.getData('text');
      if(!pasted)return;
      event.preventDefault();
      const value = onlyDigits(pasted).length >= 7 ? pasted : phoneInput.value.slice(0,phoneInput.selectionStart||0)+pasted+phoneInput.value.slice(phoneInput.selectionEnd||0);
      phoneInput.value = formatFinnishPhone(value);
      phoneInput.dispatchEvent(new Event('input',{bubbles:true}));
    });

    phoneInput.addEventListener("blur", function () {
      if (phoneInput.value === "+358 ") phoneInput.value = "";
    });
  }

  if (carInput) {
    carInput.addEventListener("input", function () {
      const oldValue = carInput.value;
      const oldStart = carInput.selectionStart || 0;
      const newValue = normalizeRegistrationNumber(oldValue);
      carInput.value = newValue;
      setCookie("car", newValue, 30);

      let newCursor = oldStart;
      if (newValue.length > oldValue.length && newValue.includes("-") && !oldValue.includes("-")) {
        newCursor += 1;
      }

      // Restore the caret in this input event. A queued frame can collapse the
      // next selection and truncate a quick replacement at maxlength.
      try {
        carInput.setSelectionRange(newCursor, newCursor);
      } catch (e) {}
    });
  }

  if (messageInput) {
    autoResizeTextarea(messageInput);
    messageInput.addEventListener("input", function () {
      autoResizeTextarea(messageInput);
    });
  }

  if (photosTrigger && photosInput) {
    photosTrigger.addEventListener("click", function () {
      photosInput.click();
    });

    photosInput.addEventListener("change", function () {
      handlePhotoSelection(photosInput.files);
    });
  }

  if (mobileStickyFormLink) {
    mobileStickyFormLink.addEventListener("click", function () {
      trackEvent("mobile_sticky_form_click", {
        page_language: i18n.lang
      });
      focusFirstUsefulField();
    });
  }

  form.addEventListener("submit", handleSubmit);
}

document.addEventListener("DOMContentLoaded", function () {
  const sliderWrap = document.querySelector("[data-testimonial-slider]");
  if (!sliderWrap) return;

  const toolbar = sliderWrap.querySelector("[data-testimonial-toolbar]");
  const track = sliderWrap.querySelector("[data-testimonial-track]");
  const prevBtn = sliderWrap.querySelector("[data-testimonial-prev]");
  const nextBtn = sliderWrap.querySelector("[data-testimonial-next]");

  if (!track || !prevBtn || !nextBtn) return;

  function getGap() {
    const styles = window.getComputedStyle(track);
    const gap = parseFloat(styles.columnGap || styles.gap || "20");
    return Number.isNaN(gap) ? 20 : gap;
  }

  function getScrollStep() {
    const firstCard = track.querySelector(".testimonial-slide");
    if (!firstCard) return track.clientWidth;
    const cardWidth = firstCard.getBoundingClientRect().width;
    return cardWidth + getGap();
  }

  function updateSliderUI() {
    const maxScrollLeft = track.scrollWidth - track.clientWidth;
    const needsSlider = maxScrollLeft > 8;

    if (toolbar) {
      toolbar.classList.toggle("is-hidden", !needsSlider);
    }

    if (!needsSlider) {
      prevBtn.disabled = true;
      nextBtn.disabled = true;
      track.scrollLeft = 0;
      return;
    }

    prevBtn.disabled = track.scrollLeft <= 4;
    nextBtn.disabled = track.scrollLeft >= maxScrollLeft - 4;
  }

  prevBtn.addEventListener("click", function () {
    track.scrollBy({ left: -getScrollStep(), behavior: "smooth" });
  });

  nextBtn.addEventListener("click", function () {
    track.scrollBy({ left: getScrollStep(), behavior: "smooth" });
  });

  track.addEventListener("scroll", updateSliderUI);
  window.addEventListener("resize", updateSliderUI);

  updateSliderUI();
});

document.addEventListener("DOMContentLoaded", function () {
  const stickyHeader = document.querySelector(".header, .site-header");
  const anchors = document.querySelectorAll('a[href^="#"]');

  anchors.forEach(function (anchor) {
    anchor.addEventListener("click", function (event) {
      const href = anchor.getAttribute("href");
      if (!href || href === "#") return;
      const target = document.querySelector(href);
      if (!target) return;
      event.preventDefault();
      const headerOffset = stickyHeader ? stickyHeader.offsetHeight + 14 : 14;
      const targetTop = target.getBoundingClientRect().top + window.scrollY - headerOffset;
      window.scrollTo({ top: Math.max(targetTop, 0), behavior: "smooth" });
    });
  });

  const revealTargets = document.querySelectorAll(".stats-strip, .section, .final-cta, .site-footer");
  revealTargets.forEach(function (el) {
    el.classList.add("reveal-section");
  });

  if ("IntersectionObserver" in window) {
    const observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          entry.target.classList.add("is-visible");
          observer.unobserve(entry.target);
        }
      });
    }, { threshold: 0.12, rootMargin: "0px 0px -40px 0px" });

    revealTargets.forEach(function (el) {
      observer.observe(el);
    });
  } else {
    revealTargets.forEach(function (el) {
      el.classList.add("is-visible");
    });
  }
});
