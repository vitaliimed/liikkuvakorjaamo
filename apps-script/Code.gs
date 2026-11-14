// Liikkuva Auto Korjaamo — Google Sheets intake and marketing dashboard
// Use your EXISTING spreadsheet. This script never creates, clears or deletes a spreadsheet.
const SITE='liikkuvakorjaamo.fi';
const ZONE='Europe/Helsinki';
const SHEET_ID='14HHvj7skyve3K9ugFFgdlobD9De-2BICFdkSZdrVHKU';
const GREEN='#23372b',ORANGE='#f97315';
const HEADERS=[
  'Статус','Доход, €','За что получили','Дата Helsinki','Имя','Телефон','Email',
  'Автомобиль','Адрес','Сообщение','Язык','Фото','Карта','Источник','UTM source',
  'UTM medium','UTM campaign','Страница','Сайт','ID заявки','Выезд, €',
  'Диагностика от, €','Канал','IP (маска сети)'
];
const SUMMARY_HEADERS=[
  'Месяц','Доход всего, €','Успешные заказы',
  'Расходы Ads, € — вручную','Клики Ads — вручную',
  'Заявки из рекламы','Успешные из рекламы',
  'Доход из рекламы, €','ROAS, ×','CR: клик → заявка',
  'CPL, €','CPA, €','Доля закрытия рекламных заявок'
];

// RUN THIS FUNCTION ONCE FROM THE SCRIPT EDITOR.
// This is the ID of your existing Google Sheet, NOT the Apps Script editor ID.
function setupExistingLiikkuva(){
  const p=PropertiesService.getScriptProperties();
  const oldSite=p.getProperty('SITE_DOMAIN');
  if(oldSite&&oldSite!==SITE)throw new Error('This script already belongs to another website');
  const b=SpreadsheetApp.openById(SHEET_ID);
  if(b.getSheetByName('__autohelp_meta'))throw new Error('AutoHelp data found: refuse to use a different business sheet');
  let marker=b.getSheetByName('__liikkuva_meta');
  if(marker&&marker.getRange('A1').getValue()&&marker.getRange('A1').getValue()!==SITE){
    throw new Error('Foreign site marker; do not overwrite');
  }
  const existing=p.getProperty('WEBHOOK_TOKEN');
  const properties={SPREADSHEET_ID:b.getId(),SITE_DOMAIN:SITE};
  if(!existing)properties.WEBHOOK_TOKEN=newToken_();
  // Configure IDs before creating the monthly tabs; never destroy existing rows.
  p.setProperties(properties);
  if(!marker)marker=b.insertSheet('__liikkuva_meta');
  marker.getRange('A1').setValue(SITE);
  b.setSpreadsheetTimeZone(ZONE);
  const month=Utilities.formatDate(new Date(),ZONE,'yyyy-MM');
  year_(b,Number(month.slice(0,4)));
  first_(b,month);
  marker.hideSheet();
  console.log('Таблица Liikkuva подключена: '+b.getUrl());
  console.log('Токен: Project settings → Script properties → WEBHOOK_TOKEN. Никому не отправляйте.');
  console.log('Далее: Deploy → New deployment → Web app. Для Cloud Run нужна ссылка /exec.');
}
// Old setup name is retained to avoid accidental creation of a second document.
function setupNewLiikkuva(){return setupExistingLiikkuva();}
function newToken_(){return Utilities.getUuid().replace(/-/g,'')+Utilities.getUuid().replace(/-/g,'');}
function openBook_(){
  const p=PropertiesService.getScriptProperties();
  if(p.getProperty('SITE_DOMAIN')!==SITE||p.getProperty('SPREADSHEET_ID')!==SHEET_ID){
    throw new Error('Run setupExistingLiikkuva first; wrong sheet binding');
  }
  // In Web App context getActiveSpreadsheet() is NOT reliable.
  const b=SpreadsheetApp.openById(SHEET_ID);
  const marker=b.getSheetByName('__liikkuva_meta');
  if(!marker||marker.getRange('A1').getValue()!==SITE)throw new Error('Wrong Liikkuva document');
  return b;
}
function doGet(){return json_({ok:true,site:SITE,service:'Liikkuva Google Sheets'});}
function doPost(event){
  let lock;
  try{
    const raw=event&&event.postData&&event.postData.contents;
    if(!raw||raw.length>100000)return json_({ok:false,error:'Invalid payload'});
    const l=JSON.parse(raw);
    const token=PropertiesService.getScriptProperties().getProperty('WEBHOOK_TOKEN');
    if(!token||token.length<32||l.script_token!==token)return json_({ok:false,error:'Unauthorized'});
    if(l.site!==SITE)return json_({ok:false,error:'Wrong site'});
    const book=openBook_();
    if(l.action==='check_connection')return json_({ok:true,site:SITE,configured:true});
    if(!/^[A-Za-z0-9_-]{16,128}$/.test(l.submission_id||''))return json_({ok:false,error:'Missing submission ID'});
    if(!l.name||!l.phone)return json_({ok:false,error:'Missing contact'});
    if(!/^20\d{2}-(0[1-9]|1[0-2])-\d{2} \d{2}:\d{2}:\d{2}$/.test(l.created_at||'')){
      return json_({ok:false,error:'Invalid Helsinki timestamp'});
    }
    lock=LockService.getScriptLock();
    if(!lock.tryLock(20000))return json_({ok:false,error:'Busy; retry'});
    const month=l.created_at.slice(0,7);
    year_(book,Number(month.slice(0,4)));
    const sheet=month_(book,month),last=sheet.getLastRow();
    // Strong idempotency: Cloud Tasks may retry delivery.
    if(last>1&&sheet.getRange(2,20,last-1,1).createTextFinder(l.submission_id).matchEntireCell(true).findNext()){
      return json_({ok:true,sheet:month,message:'Already saved'});
    }
    if(last>=sheet.getMaxRows()){
      const oldMax=sheet.getMaxRows();sheet.insertRowsAfter(oldMax,500);
      status_(sheet,oldMax+1,500);channelValidation_(sheet,oldMax+1,500);
    }
    sheet.appendRow([
      'В процессе','', '',safe_(l.created_at),safe_(l.name),safe_(l.phone),
      safe_(l.email),safe_(l.car),safe_(l.address),safe_(l.message),safe_(l.language),
      safe_((l.photo_urls||[]).join('\n')),safe_(l.maps_url),safe_(l.source),
      safe_(l.utm_source),safe_(l.utm_medium),safe_(l.utm_campaign),
      safe_(l.page_url),SITE,l.submission_id,Number(l.visit_price_eur)||75,
      Number(l.diagnostic_from_eur)||50,channel_(l.utm_medium,l.source),maskIP_(l.ip)
    ]);
    SpreadsheetApp.flush();
    const current=Utilities.formatDate(new Date(),ZONE,'yyyy-MM');
    year_(book,Number(current.slice(0,4)));
    first_(book,current);
    return json_({ok:true,sheet:month});
  }catch(e){
    // Error messages must never expose token, customer data or spreadsheet IDs.
    console.error('Liikkuva Sheets: request failed ('+(e&&e.name||'unknown')+')');
    return json_({ok:false,error:'Storage configuration error'});
  }finally{if(lock&&lock.hasLock())lock.releaseLock();}
}
function safe_(v){
  const s=String(v==null?'':v).slice(0,20000);
  return /^\s*[=+@-]/.test(s)?"'"+s:s;
}
// Client IPs are personal data. For anti-abuse analysis, retain only a
// non-identifying network prefix in the marketing spreadsheet, never full IP.
// The exact client IP remains in restricted Firestore lead records.
function maskIP_(value){
  const ip=String(value||'').trim();
  if(/^\d{1,3}(\.\d{1,3}){3}$/.test(ip)){
    const parts=ip.split('.').map(Number);
    if(parts.some(n=>n<0||n>255))return '';
    return parts.slice(0,3).join('.')+'.0/24';
  }
  if(/^[0-9a-f:]+$/i.test(ip)&&ip.includes(':')){
    const chunks=ip.toLowerCase().split(':').filter(Boolean);
    if(chunks.length>=3)return chunks.slice(0,3).join(':')+'::/48';
  }
  return '';
}
function channel_(medium,source){
  const m=String(medium||'').toLowerCase().trim().replace(/[\s-]/g,'_');
  const s=String(source||'').toLowerCase().trim().replace(/[\s-]/g,'_');
  if(/^(cpc|ppc|paid|paid_search|paid_social|social_paid|display|cpm|cpv|retargeting)$/.test(m)||
     /^(google_ads|facebook_ads|meta_ads|instagram_ads)$/.test(s))return 'Реклама';
  if(/^(organic|organic_search|organic_social|seo)$/.test(m))return 'Органика';
  return 'Не определён';
}
function status_(s,row,count){
  if(count<1)return;
  const rule=SpreadsheetApp.newDataValidation()
    .requireValueInList(['В процессе','Успешный','Спам','Отказ'],true)
    .setAllowInvalid(false).build();
  s.getRange(row,1,count,1).setDataValidation(rule);
}
function channelValidation_(s,row,count){
  if(count<1)return;
  s.getRange(row,23,count,1).setDataValidation(
    SpreadsheetApp.newDataValidation()
      .requireValueInList(['Реклама','Органика','Не определён'],true)
      .setAllowInvalid(false).build());
}
function month_(book,name){
  let s=book.getSheetByName(name);
  if(!s){
    s=book.insertSheet(name);
    s.getRange(1,1,1,HEADERS.length).setValues([HEADERS]);
    s.getRange(1,1,1,HEADERS.length).setBackground(GREEN).setFontColor('#fff').setFontWeight('bold');
    s.setFrozenRows(1);
    s.setColumnWidths(1,3,150);
    s.setColumnWidth(4,180);
    s.setColumnWidths(5,5,180);
    s.setColumnWidth(10,340);
    s.setColumnWidth(23,160);
    s.setColumnWidth(24,180);
    s.hideColumns(20); // Internal deduplication ID
    status_(s,2,s.getMaxRows()-1);
    channelValidation_(s,2,s.getMaxRows()-1);
    s.getRange(2,2,s.getMaxRows()-1,1).setNumberFormat('#,##0.00 "€"');
    s.getRange(1,1,s.getMaxRows(),HEADERS.length).createFilter();
    return s;
  }
  // Keep historical customer cells and the deduplication column untouched.
  // Add the new, privacy-minimized IP network only in an empty X column.
  if(s.getMaxColumns()<24)s.insertColumnsAfter(s.getMaxColumns(),24-s.getMaxColumns());
  const channelHeader=s.getRange('W1').getValue();
  if(channelHeader===''){
    s.getRange('W1').setValue('Канал').setBackground(GREEN).setFontColor('#fff').setFontWeight('bold');
    const count=s.getLastRow()-1;
    if(count>0){
      const data=s.getRange(2,14,count,3).getValues(); // N=source O=UTM source P=UTM medium
      s.getRange(2,23,count,1).setValues(data.map(r=>[channel_(r[2],r[0])]));
    }
  }else if(channelHeader!=='Канал'){
    throw new Error('Column W has user data; cannot repurpose it automatically');
  }
  const ipHeader=s.getRange('X1').getValue();
  if(ipHeader===''){
    const last=s.getLastRow();
    if(last>1&&s.getRange(2,24,last-1,1).getValues().some(row=>row[0]!=='')){
      throw new Error('Column X already contains user data; do not overwrite');
    }
    s.getRange('X1').setValue('IP (маска сети)').setBackground(GREEN).setFontColor('#fff').setFontWeight('bold');
    s.setColumnWidth(24,180);
  }else if(ipHeader!=='IP (маска сети)'){
    throw new Error('Column X has user data; cannot repurpose automatically');
  }
  channelValidation_(s,2,s.getMaxRows()-1);
  return s;
}
function year_(b,y){
  for(let m=1;m<=12;m++)month_(b,y+'-'+String(m).padStart(2,'0'));
  const name='Доход '+y;
  let s=b.getSheetByName(name);
  if(!s)s=b.insertSheet(name);
  // Never overwrite existing user-created summary content in D/E.
  for(const col of [4,5]){
    const value=s.getRange(1,col).getValue();
    if(value&&value!==SUMMARY_HEADERS[col-1]){
      throw new Error('Income sheet already uses '+(col===4?'D':'E')+' — manually review before upgrading');
    }
  }
  // If this schema is already initialized, don't rewrite formulas on each webhook.
  if(s.getRange('M1').getValue()===SUMMARY_HEADERS[12])return;
  s.getRange(1,1,1,SUMMARY_HEADERS.length).setValues([SUMMARY_HEADERS]);
  s.getRange(1,1,1,SUMMARY_HEADERS.length).setBackground(GREEN).setFontColor('#fff').setFontWeight('bold');
  s.getRange('D1:E1').setBackground(ORANGE).setFontColor('#23372b');
  for(let m=1;m<=12;m++){
    const row=m+1,name=y+'-'+String(m).padStart(2,'0'),q="'"+name+"'!";
    s.getRange(row,1).setValue(name);
    const formulas=[
      [2,'=SUMIF('+q+'A2:A,"Успешный",'+q+'B2:B)'],
      [3,'=COUNTIF('+q+'A2:A,"Успешный")'],
      [6,'=COUNTIFS('+q+'W2:W,"Реклама",'+q+'A2:A,"<>Спам")'],
      [7,'=COUNTIFS('+q+'W2:W,"Реклама",'+q+'A2:A,"Успешный")'],
      [8,'=SUMIFS('+q+'B2:B,'+q+'W2:W,"Реклама",'+q+'A2:A,"Успешный")'],
      [9,'=IF(D'+row+'>0,H'+row+'/D'+row+',"")'],
      [10,'=IF(E'+row+'>0,F'+row+'/E'+row+',"")'],
      [11,'=IF(F'+row+'>0,D'+row+'/F'+row+',"")'],
      [12,'=IF(G'+row+'>0,D'+row+'/G'+row+',"")'],
      [13,'=IF(F'+row+'>0,G'+row+'/F'+row+',"")']
    ];
    formulas.forEach(pair=>s.getRange(row,pair[0]).setFormula(pair[1]));
    // D/E are MANUAL inputs and are deliberately never cleared.
  }
  s.getRange('A15').setValue('Итого');
  for(const c of ['B','C','D','E','F','G','H']){
    s.getRange(c+'15').setFormula('=SUM('+c+'2:'+c+'13)');
  }
  const totals={
    I:'=IF(D15>0,H15/D15,"")',
    J:'=IF(E15>0,F15/E15,"")',
    K:'=IF(F15>0,D15/F15,"")',
    L:'=IF(G15>0,D15/G15,"")',
    M:'=IF(F15>0,G15/F15,"")'
  };
  Object.keys(totals).forEach(c=>s.getRange(c+'15').setFormula(totals[c]));
  s.getRange('A15:M15').setBackground(GREEN).setFontColor('#fff').setFontWeight('bold');
  for(const c of ['B','D','H','K','L'])s.getRange(c+'2:'+c+'15').setNumberFormat('#,##0.00 "€"');
  s.getRange('I2:I15').setNumberFormat('0.00"x"');
  for(const c of ['J','M'])s.getRange(c+'2:'+c+'15').setNumberFormat('0.0%');
  s.getRange('E2:G15').setNumberFormat('#,##0');
  s.getRange('D2:E13').setBackground('#f4f5f2'); // Manual-input cells
  s.setFrozenRows(1);
  s.setColumnWidths(1,13,188);
  s.setColumnWidth(4,205);s.setColumnWidth(5,205);
  s.getRange('A17').setValue('D/E: вносите расходы и клики из Google Ads / Meta Ads вручную.');
  s.getRange('A18').setValue('CR = заявки из рекламы / рекламные клики. ROAS = оплаченный рекламный доход / расходы на рекламу.');
  s.getRange('A19').setValue('Рекламные заявки определяются по UTM medium; при отсутствии UTM используйте ручную классификацию в W.');
}
function first_(b,name){
  b.setActiveSheet(b.getSheetByName('Доход '+name.slice(0,4)));b.moveActiveSheet(1);
  b.setActiveSheet(month_(b,name));b.moveActiveSheet(1);
}
function showCurrentMonth(){
  const b=openBook_(),m=Utilities.formatDate(new Date(),ZONE,'yyyy-MM');
  year_(b,Number(m.slice(0,4)));first_(b,m);
}
function json_(data){
  return ContentService.createTextOutput(JSON.stringify(data))
    .setMimeType(ContentService.MimeType.JSON);
}
