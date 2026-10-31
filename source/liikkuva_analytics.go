package main

import (
	"encoding/json"
	"strings"
)

type analyticsEventSetting struct {
	Name, Label, Description string
	Enabled                  bool
}

func analyticsEventCatalog(s SiteSettings) []analyticsEventSetting {
	events := []analyticsEventSetting{
		{Name: "generate_lead", Label: "Заявка принята", Description: "Только после успешного сохранения заявки. Основная конверсия; один раз на заявку."},
		{Name: "lead_thankyou_view", Label: "Страница благодарности", Description: "Просмотр подтверждения после отправки; не отдельная основная конверсия."},
		{Name: "lead_form_view", Label: "Просмотр формы", Description: "Форма появилась в области экрана."},
		{Name: "lead_form_start", Label: "Начало заполнения", Description: "Первое взаимодействие с полем формы."},
		{Name: "lead_form_open", Label: "Переход к форме", Description: "Нажатие кнопки заявки в шапке, услуге или другом блоке."},
		{Name: "lead_submit_attempt", Label: "Попытка отправки", Description: "Нажатие отправки; ещё не является принятой заявкой."},
		{Name: "lead_submit_success", Label: "Успешный ответ формы", Description: "Технический ответ сервера. Не отмечайте второй основной конверсией."},
		{Name: "lead_validation_error", Label: "Ошибка заполнения", Description: "Название ошибочного поля, без его содержимого."},
		{Name: "lead_submit_error", Label: "Отказ сервера", Description: "Форма не принята сервером."},
		{Name: "lead_submit_network_error", Label: "Ошибка соединения", Description: "Не удалось получить ответ при отправке."},
		{Name: "lead_photo_upload", Label: "Добавление фото", Description: "Количество и результат проверки; без файлов и имён файлов."},
		{Name: "lead_use_location_click", Label: "Определение местоположения", Description: "Нажатие кнопки геолокации; без координат."},
		{Name: "lead_map_open", Label: "Выбор адреса на карте", Description: "Открытие карты для указания места автомобиля."},
		{Name: "lead_map_point_confirmed", Label: "Подтверждение места", Description: "Выбрана точка на карте; без адреса и координат."},
		{Name: "phone_click", Label: "Нажатие телефона", Description: "Переход к звонку; без номера телефона."},
		{Name: "whatsapp_click", Label: "Переход в WhatsApp", Description: "Нажатие ссылки WhatsApp; без номера и текста."},
		{Name: "email_click", Label: "Нажатие почты", Description: "Открытие почтовой программы; без адреса."},
		{Name: "mobile_sticky_form_click", Label: "Мобильная кнопка заявки", Description: "Нажатие закреплённой кнопки на телефоне."},
		{Name: "navigation_click", Label: "Навигация", Description: "Переход между разделами сайта."},
		{Name: "service_select", Label: "Выбор услуги", Description: "Номер выбранной карточки услуги, без текста заявки."},
		{Name: "view_pricing", Label: "Просмотр цен", Description: "Блок цен появился в области экрана."},
		{Name: "view_service_area", Label: "Просмотр зоны работы", Description: "Блок карты появился в области экрана."},
		{Name: "service_area_map_interaction", Label: "Действия с картой", Description: "Масштабирование и перемещение обзорной карты."},
		{Name: "faq_open", Label: "Открытие вопроса", Description: "Номер открытого вопроса."},
		{Name: "theme_change", Label: "Переключение оформления", Description: "Выбрана светлая или тёмная тема."},
		{Name: "language_switch", Label: "Переключение версии", Description: "Переход между FI и EN."},
		{Name: "scroll_depth", Label: "Глубина просмотра", Description: "Достигнуты 25, 50, 75 или 90 процентов страницы."},
	}
	disabled := map[string]bool{}
	for _, name := range s.AnalyticsDisabledEvents {
		disabled[name] = true
	}
	for i := range events {
		events[i].Enabled = !s.AnalyticsEventsConfigured || !disabled[events[i].Name]
	}
	return events
}

func disabledAnalyticsEvents(enabled []string) []string {
	selected := map[string]bool{}
	for _, name := range enabled {
		selected[name] = true
	}
	var disabled []string
	for _, event := range analyticsEventCatalog(SiteSettings{}) {
		if !selected[event.Name] {
			disabled = append(disabled, event.Name)
		}
	}
	return disabled
}

func ga4ID(s SiteSettings) string {
	if s.AnalyticsDisabled {
		return ""
	}
	return strings.TrimSpace(s.GA4MeasurementID)
}

func trackingConfig(s SiteSettings) map[string]any {
	s = liikkuvaSettings(s)
	enabled := map[string]bool{}
	for _, event := range analyticsEventCatalog(s) {
		enabled[event.Name] = event.Enabled
	}
	return map[string]any{"ga4MeasurementId": ga4ID(s), "primaryGoogleTagId": primaryGoogleTagID(s),
		"googleAdsTagId": s.GoogleAdsTagID, "googleAdsLeadConversionLabel": s.GoogleAdsLeadConversionLabel,
		"metaPixelId": s.MetaPixelID, "linkedinPartnerId": s.LinkedInPartnerID,
		"linkedinLeadConversionID": s.LinkedInLeadConversionID, "tiktokPixelId": s.TikTokPixelID, "events": enabled}
}

func liikkuvaTrackingHTML(body string, s SiteSettings) string {
	config, _ := json.Marshal(trackingConfig(s)) // encoding/json escapes HTML delimiters.
	return strings.Replace(body, "</head>", `<script>window.LIIKKUVA_TRACKING=`+string(config)+`;</script></head>`, 1)
}
