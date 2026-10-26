const SITE='liikkuvakorjaamo.fi', ZONE='Europe/Helsinki';
const HEADERS=['Статус','Доход, €','За что получили','Дата Helsinki','Имя','Телефон','Email','Автомобиль','Адрес','Сообщение','Язык','Фото','Карта','Источник','UTM source','UTM medium','UTM campaign','Страница','Сайт','ID заявки','Выезд, €','Диагностика от, €'];
function setupNewLiikkuva(){
 const p=PropertiesService.getScriptProperties();
 if(p.getProperty('SPREADSHEET_ID')){console.log(openBook_().getUrl());return;}
 const b=SpreadsheetApp.create('Liikkuva Auto Korjaamo — заявки и доход');
 b.setSpreadsheetTimeZone(ZONE);b.setSpreadsheetLocale('en_US');
 const marker=b.getSheets()[0];marker.setName('__liikkuva_meta');marker.getRange('A1').setValue(SITE);
 p.setProperties({SPREADSHEET_ID:b.getId(),SITE_DOMAIN:SITE,WEBHOOK_TOKEN:Utilities.getUuid().replace(/-/g,'')+Utilities.getUuid().replace(/-/g,'')});
 const month=Utilities.formatDate(new Date(),ZONE,'yyyy-MM');year_(b,Number(month.slice(0,4)));first_(b,month);marker.hideSheet();
 console.log('Новая таблица: '+b.getUrl());
 console.log('WEBHOOK_TOKEN: Project settings → Script properties. Не отправляйте токен в чат.');
}
function openBook_(){
 const p=PropertiesService.getScriptProperties();
 if(p.getProperty('SITE_DOMAIN')!==SITE)throw new Error('Wrong site');
 const b=SpreadsheetApp.openById(p.getProperty('SPREADSHEET_ID'));
 const marker=b.getSheetByName('__liikkuva_meta');
 if(!marker||marker.getRange('A1').getValue()!==SITE)throw new Error('Wrong document');
 return b;
}
function doGet(){return json_({ok:true,site:SITE});}
function doPost(event){
 let lock;
 try{
  const raw=event&&event.postData&&event.postData.contents;
  if(!raw||raw.length>100000)return json_({ok:false,error:'Invalid payload'});
  const l=JSON.parse(raw),token=PropertiesService.getScriptProperties().getProperty('WEBHOOK_TOKEN');
  if(!token||token.length<32||l.script_token!==token)return json_({ok:false,error:'Unauthorized'});
  if(l.site!==SITE)return json_({ok:false,error:'Wrong site'});
  const book=openBook_();
  if(l.action==='check_connection')return json_({ok:true,site:SITE,configured:true});
  if(!/^[A-Za-z0-9_-]{16,128}$/.test(l.submission_id||''))return json_({ok:false,error:'Missing submission ID'});
  if(!l.name||!l.phone)return json_({ok:false,error:'Missing contact'});
  if(!/^20\d{2}-(0[1-9]|1[0-2])-\d{2} \d{2}:\d{2}:\d{2}$/.test(l.created_at||''))return json_({ok:false,error:'Invalid date'});
  lock=LockService.getScriptLock();if(!lock.tryLock(20000))return json_({ok:false,error:'Busy; retry'});
  const month=l.created_at.slice(0,7);year_(book,Number(month.slice(0,4)));
  const sheet=book.getSheetByName(month),last=sheet.getLastRow();
  if(last>1&&sheet.getRange(2,20,last-1,1).createTextFinder(l.submission_id).matchEntireCell(true).findNext())return json_({ok:true,sheet:month,message:'Already saved'});
  if(last>=sheet.getMaxRows()){const n=sheet.getMaxRows();sheet.insertRowsAfter(n,500);status_(sheet,n+1,500);}
  sheet.appendRow(['В процессе','','',safe_(l.created_at),safe_(l.name),safe_(l.phone),safe_(l.email),safe_(l.car),safe_(l.address),safe_(l.message),safe_(l.language),safe_((l.photo_urls||[]).join('\n')),safe_(l.maps_url),safe_(l.source),safe_(l.utm_source),safe_(l.utm_medium),safe_(l.utm_campaign),safe_(l.page_url),SITE,l.submission_id,Number(l.visit_price_eur)||70,Number(l.diagnostic_from_eur)||50]);
  SpreadsheetApp.flush();
  const current=Utilities.formatDate(new Date(),ZONE,'yyyy-MM');year_(book,Number(current.slice(0,4)));first_(book,current);
  return json_({ok:true,sheet:month});
 }catch(e){return json_({ok:false,error:'Storage configuration error'});}
 finally{if(lock&&lock.hasLock())lock.releaseLock();}
}
function safe_(v){const s=String(v==null?'':v).slice(0,20000);return /^\s*[=+@-]/.test(s)?"'"+s:s;}
function status_(s,row,count){s.getRange(row,1,count,1).setDataValidation(SpreadsheetApp.newDataValidation().requireValueInList(['В процессе','Успешный','Спам','Отказ'],true).setAllowInvalid(false).build());}
function month_(b,name){
 let s=b.getSheetByName(name);if(s)return s; // NEVER clear or replace existing monthly data.
 s=b.insertSheet(name);s.getRange(1,1,1,HEADERS.length).setValues([HEADERS]).setBackground('#202c29').setFontColor('#fff').setFontWeight('bold');s.setFrozenRows(1);
 s.setColumnWidths(1,3,150);s.setColumnWidth(4,180);s.setColumnWidths(5,5,180);s.setColumnWidth(10,340);s.hideColumns(20);
 status_(s,2,s.getMaxRows()-1);s.getRange(2,2,s.getMaxRows()-1,1).setNumberFormat('#,##0.00 "€"');s.getRange(1,1,s.getMaxRows(),HEADERS.length).createFilter();return s;
}
function year_(b,y){
 for(let m=1;m<=12;m++)month_(b,y+'-'+String(m).padStart(2,'0'));
 if(b.getSheetByName('Доход '+y))return;
 const s=b.insertSheet('Доход '+y);s.getRange('A1:C1').setValues([['Месяц','Доход успешных заявок, €','Успешные заявки']]).setBackground('#cf512b').setFontColor('#fff').setFontWeight('bold');
 for(let m=1;m<=12;m++){const name=y+'-'+String(m).padStart(2,'0');s.getRange(m+1,1).setValue(name);s.getRange(m+1,2).setFormula('=SUMIF(\''+name+'\'!A2:A,"Успешный",\''+name+'\'!B2:B)');s.getRange(m+1,3).setFormula('=COUNTIF(\''+name+'\'!A2:A,"Успешный")');}
 s.getRange('A15').setValue('Итого');s.getRange('B15').setFormula('=SUM(B2:B13)');s.getRange('C15').setFormula('=SUM(C2:C13)');s.getRange('B2:B15').setNumberFormat('#,##0.00 "€"');s.setColumnWidths(1,3,220);s.setFrozenRows(1);
}
function first_(b,name){b.setActiveSheet(b.getSheetByName('Доход '+name.slice(0,4)));b.moveActiveSheet(1);b.setActiveSheet(month_(b,name));b.moveActiveSheet(1);}
function showCurrentMonth(){const b=openBook_(),m=Utilities.formatDate(new Date(),ZONE,'yyyy-MM');year_(b,Number(m.slice(0,4)));first_(b,m);}
function json_(object){return ContentService.createTextOutput(JSON.stringify(object)).setMimeType(ContentService.MimeType.JSON);}
