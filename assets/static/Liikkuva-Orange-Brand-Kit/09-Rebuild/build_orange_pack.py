from pathlib import Path
import copy, csv, gzip, io, json, re, zipfile, hashlib, subprocess
import xml.etree.ElementTree as ET
from collections import Counter
from PIL import Image, ImageDraw, ImageFont
import cairosvg
from reportlab.pdfgen import canvas
from reportlab.lib.utils import ImageReader
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from pypdf import PdfReader

ROOT=Path.cwd()
OUT=ROOT/'Liikkuva-Orange-Brand-Kit'
OUT.mkdir(exist_ok=True)
ORANGE='#F97316'; GREEN='#123F32'; DARK='#17191C'; INK='#001A4B'; WHITE='#FFFFFF'
SOURCE=ROOT/'original/extracted/liikkuva-logo-pack/assets/static/img/logo-light.svg'
if (Path(__file__).parent/'source-master.svg').exists():SOURCE=Path(__file__).parent/'source-master.svg'
ET.register_namespace('', 'http://www.w3.org/2000/svg')
root=ET.parse(SOURCE).getroot()
group=list(root)[1]
paths=list(group)
def y_pos(p): return float(re.search(r'translate\([^,]+,([^\)]+)',p.attrib['transform']).group(1))
def artwork(fg,kind='full',mono=None):
    g=copy.deepcopy(group)
    for p in list(g):
        y=y_pos(p)
        if (kind=='mark' and y>=525) or (kind=='wordmark' and y<525):g.remove(p);continue
        p.set('fill', mono or (ORANGE if p.get('fill')=='#0755D8' else fg))
    return ET.tostring(g,encoding='unicode')
def svg(inner,w=2000,h=720,bg=None,viewbox=None):
    box=viewbox or f'0 0 {w} {h}'
    rect=f'<rect width="{w}" height="{h}" fill="{bg}"/>' if bg else ''
    return f'<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="{box}" role="img" aria-label="liikkuva auto korjaamo"><title>liikkuva auto korjaamo</title>{rect}{inner}</svg>'
def save(path,data):
    path=OUT/path;path.parent.mkdir(parents=True,exist_ok=True)
    if isinstance(data,str):path.write_text(data,encoding='utf-8')
    else:path.write_bytes(data)
    return path
def render(doc,w):
    return Image.open(io.BytesIO(cairosvg.svg2png(bytestring=doc.encode(),output_width=w))).convert('RGBA')
def write_raster(im,path,fmt):
    path=OUT/path;path.parent.mkdir(parents=True,exist_ok=True)
    if fmt=='PNG':im.save(path,optimize=True,dpi=(300,300))
    elif fmt=='WEBP':im.save(path,format='WEBP',lossless=True,method=6)
    elif fmt=='AVIF':im.save(path,format='AVIF',quality=95,subsampling='4:4:4',speed=6)
    elif fmt=='JPEG':im.convert('RGB').save(path,quality=96,subsampling=0,dpi=(300,300))
    elif fmt=='TIFF':im.save(path,compression='tiff_lzw',dpi=(300,300))
    elif fmt=='BMP':im.convert('RGB').save(path)
    elif fmt=='GIF':im.convert('RGB').quantize(colors=256).save(path)
def vectors(name,doc,folder='01-Vector',all_formats=True):
    save(Path(folder)/(name+'.svg'),doc)
    save(Path(folder)/(name+'.svgz'),gzip.compress(doc.encode(),mtime=0))
    save(Path(folder)/(name+'.pdf'),cairosvg.svg2pdf(bytestring=doc.encode()))
    save(Path(folder)/(name+'.eps'),cairosvg.svg2eps(bytestring=doc.encode()))
    if all_formats:save(Path(folder)/(name+'.ps'),cairosvg.svg2ps(bytestring=doc.encode()))
    for ext in ['emf','wmf']:
        subprocess.run(['inkscape',str(OUT/folder/(name+'.svg')),'--export-type='+ext,'--export-filename='+str(OUT/folder/(name+'.'+ext))],check=True,capture_output=True)

themes=[('white',INK,WHITE),('dark',WHITE,DARK),('green',WHITE,GREEN)]
docs={}
for name,fg,bg in themes:
    docs[name]=svg(artwork(fg),bg=bg)
docs['transparent-light']=svg(artwork(INK))
docs['transparent-dark']=svg(artwork(WHITE))
for name,doc in docs.items():
    vectors('logo-orange-'+name,doc)
    for width in [400,800,1600,3200]:
        im=render(doc,width)
        for ext,fmt in [('png','PNG'),('webp','WEBP')]:
            write_raster(im,Path('02-Raster')/name/f'logo-orange-{name}-{width}.{ext}',fmt)
        if width==1600:
            write_raster(im,Path('02-Raster')/name/f'logo-orange-{name}-{width}.avif','AVIF')
            if not name.startswith('transparent'):
                for ext,fmt in [('jpg','JPEG'),('bmp','BMP'),('gif','GIF')]:
                    write_raster(im,Path('02-Raster')/name/f'logo-orange-{name}-{width}.{ext}',fmt)
    im=render(doc,4000)
    write_raster(im,Path('03-Print')/f'logo-orange-{name}-4000px-300dpi.tif','TIFF')
    print('Logo exports:',name,flush=True)

for name,fg,bg in themes:
    for kind,w,h,tx,ty in [('mark',1800,520,-70,-20),('wordmark',2000,240,0,-500)]:
        doc=svg(f'<g transform="translate({tx} {ty})">{artwork(fg,kind)}</g>',w,h,bg)
        vectors(f'{kind}-orange-{name}',doc,'04-Additional-Lockups',False)
        for width in [1000,2400]:write_raster(render(doc,width),Path('04-Additional-Lockups')/f'{kind}-orange-{name}-{width}.png','PNG')
for name,color in [('black','#000000'),('white',WHITE)]:
    doc=svg(artwork(color,mono=color))
    vectors('logo-monochrome-'+name,doc,'03-Print',False)
    write_raster(render(doc,3200),Path('03-Print')/f'logo-monochrome-{name}-3200.png','PNG')

# Large, unambiguous spanner. The full car logo is deliberately not shrunk into a favicon.
WRENCH='M22 7 C31 6 36 15 33 23 L53 43 C60 50 50 60 43 53 L23 33 C15 36 6 31 7 22 L16 25 L25 16 Z'
WRENCH_ARROW='M22 7 C31 6 36 15 33 23 L48 38 L53 33 L57 57 L33 53 L38 48 L23 33 C15 36 6 31 7 22 L16 25 L25 16 Z'
def icon_doc(bg,maskable=False,arrow=False,transparent=False):
    path=WRENCH_ARROW if arrow else WRENCH
    # Maskable icon keeps all artwork inside the center safe circle (r=25.6).
    shape=f'<path d="{path}" fill="{ORANGE}"/>'
    if not arrow:shape+=f'<circle cx="48" cy="48" r="3.1" fill="{bg}"/>'
    if maskable:shape=f'<g transform="translate(8 8) scale(.75)">{shape}</g>'
    rect='' if transparent else f'<rect width="64" height="64" rx="{0 if maskable else 13}" fill="{bg}"/>'
    return svg(rect+shape,64,64)

icons={}
sizes=[16,24,32,48,64,96,128,180,192,256,512,1024]
for name,fg,bg in themes:
    doc=icon_doc(bg);icons[name]=doc
    folder=Path('05-Icons')/name
    vectors('icon-wrench-'+name,doc,str(folder),False)
    for size in sizes:
        write_raster(render(doc,size),folder/f'icon-{size}.png','PNG')
    large=render(doc,1024)
    large.save(OUT/folder/'icon.icns',format='ICNS',sizes=[(16,16),(32,32),(64,64),(128,128),(256,256),(512,512),(1024,1024)])
    render(doc,256).save(OUT/folder/'favicon.ico',sizes=[(16,16),(24,24),(32,32),(48,48),(64,64),(128,128),(256,256)])
    write_raster(large,folder/'icon-1024.webp','WEBP')
    mask=icon_doc(bg,maskable=True)
    save(folder/'icon-maskable.svg',mask)
    write_raster(render(mask,512),folder/'icon-maskable-512.png','PNG')
    # Apple touch artwork has an opaque square background; OS controls corner rounding.
    apple=doc.replace('rx="13"','rx="0"')
    for size in [120,152,167,180]:write_raster(render(apple,size),folder/f'apple-touch-icon-{size}.png','PNG')
    alt=icon_doc(bg,arrow=True)
    save(folder/'alternative-wrench-arrow.svg',alt)
    write_raster(render(alt,512),folder/'alternative-wrench-arrow-512.png','PNG')
    print('Icon exports:',name,flush=True)

for name,fg,bg in themes:
    for label,w,h in [('avatar',1080,1080),('square',1080,1080),('portrait',1080,1350),('story',1080,1920),('landscape',1200,630),('presentation',1920,1080)]:
        if label=='avatar':
            inner=f'<g transform="scale(16.875)">{icon_doc(bg,maskable=True).split("</title>")[1].split("</svg>")[0]}</g>'
        else:
            width=w*.82;scale=width/2000;y=(h-720*scale)/2
            inner=f'<g transform="translate({w*.09:.2f} {y:.2f}) scale({scale:.6f})">{artwork(fg)}</g>'
        doc=svg(inner,w,h,bg)
        save(Path('06-Social')/name/f'{label}-{w}x{h}.svg',doc)
        im=render(doc,w)
        for ext,fmt in [('png','PNG'),('jpg','JPEG')]:write_raster(im,Path('06-Social')/name/f'{label}-{w}x{h}.{ext}',fmt)

# Ready-to-use web subset. Paths are relative to this folder.
web=Path('07-Web-Ready')
for name in ['white','dark','green','transparent-light','transparent-dark']:
    save(web/f'logo-{name}.svg',docs[name])
    write_raster(render(docs[name],800),web/f'logo-{name}.png','PNG')
    write_raster(render(docs[name],800),web/f'logo-{name}.webp','WEBP')
save(web/'favicon.svg',icons['dark'])
for src,dst in [('favicon.ico','favicon.ico'),('icon-192.png','icon-192.png'),('icon-512.png','icon-512.png'),('icon-maskable-512.png','icon-maskable-512.png'),('apple-touch-icon-180.png','apple-touch-icon.png')]:
    save(web/dst,(OUT/'05-Icons/dark'/src).read_bytes())
manifest={'name':'liikkuva auto korjaamo','short_name':'liikkuva','start_url':'/','display':'standalone','background_color':DARK,'theme_color':ORANGE,'icons':[{'src':'icon-192.png','sizes':'192x192','type':'image/png','purpose':'any'},{'src':'icon-512.png','sizes':'512x512','type':'image/png','purpose':'any'},{'src':'icon-maskable-512.png','sizes':'512x512','type':'image/png','purpose':'maskable'}]}
save(web/'site.webmanifest',json.dumps(manifest,indent=2))
save(web/'example.html','''<!doctype html><html lang="ru"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Пример подключения логотипа</title><link rel="icon" href="favicon.ico" sizes="any"><link rel="icon" href="favicon.svg" type="image/svg+xml"><link rel="apple-touch-icon" href="apple-touch-icon.png"><link rel="manifest" href="site.webmanifest"><style>body{margin:0;padding:40px;background:#fff}.logo{width:min(480px,100%);height:auto}</style><img class="logo" src="logo-transparent-light.svg" width="2000" height="720" alt="liikkuva auto korjaamo"></html>''')

palette={'orange':ORANGE,'forest_green':GREEN,'charcoal':DARK,'navy':INK,'white':WHITE}
save(Path('08-Brand-Guide')/'palette.json',json.dumps(palette,indent=2))
save(Path('08-Brand-Guide')/'palette.css',':root {\n'+''.join(f'  --liikkuva-{k.replace("_","-")}: {v};\n' for k,v in palette.items())+'}\n')
save(Path('08-Brand-Guide')/'palette.gpl','GIMP Palette\nName: Liikkuva Orange\nColumns: 5\n#\n'+''.join(f'{int(v[1:3],16):3} {int(v[3:5],16):3} {int(v[5:7],16):3}  {k}\n' for k,v in palette.items()))

FONT='/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf'
BOLD='/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf'
pdfmetrics.registerFont(TTFont('DV',FONT));pdfmetrics.registerFont(TTFont('DVB',BOLD))
# Preview assembled from the actual vector exports.
preview=Image.new('RGB',(1800,1420),'#ECEEEB');d=ImageDraw.Draw(preview)
def text(x,y,s,size=26,color=DARK,bold=False):d.text((x,y),s,font=ImageFont.truetype(BOLD if bold else FONT,size),fill=color)
text(60,40,'liikkuva auto korjaamo',44,bold=True)
text(60,103,'ORANGE EDITION  /  LOGO & ICON SYSTEM',20,color='#68736D')
for i,(name,fg,bg) in enumerate(themes):
    x=40+i*580;y=170
    d.rounded_rectangle((x,y,x+560,y+450),24,fill=bg)
    text(x+28,y+25,{'white':'БЕЛЫЙ ФОН','dark':'ТЁМНЫЙ ФОН','green':'ЗЕЛЁНЫЙ ФОН'}[name],18,color=fg)
    im=render(docs[name],520);preview.paste(im,(x+20,y+150),im)
    text(x+28,y+395,bg,18,color=fg)
text(60,680,'Простая favicon: гаечный ключ',32,bold=True)
text(60,733,'Без текста. Крупный силуэт. Три фона и отдельный вариант ключ + стрелка.',22,color='#68736D')
for i,(name,fg,bg) in enumerate(themes):
    x=60+i*280
    im=render(icons[name],170);preview.paste(im,(x,795),im)
    text(x,985,{'white':'Белая','dark':'Тёмная','green':'Зелёная'}[name],20)
text(980,795,'Реальный размер',23,bold=True)
for i,size in enumerate([16,32,48,64]):
    x=990+i*165
    im=render(icons['dark'],size);preview.paste(im,(x,855),im)
    text(x,940,f'{size} px',18)
text(980,1010,'Вариант с направлением',22)
im=render(icon_doc(DARK,arrow=True),96);preview.paste(im,(990,1060),im)
text(60,1130,'Оранжевый '+ORANGE+'     Зелёный '+GREEN,25,bold=True)
text(60,1190,'SVG · SVGZ · PDF · EPS · PS · PNG · WebP · AVIF',24)
text(60,1232,'JPG · TIFF · BMP · GIF · ICO · ICNS · EMF · WMF',24)
text(60,1320,'Веб • Соцсети • Презентации • Документы • Печать',22,color='#68736D')
preview.save(OUT/'PREVIEW.png',optimize=True)

guide=OUT/'08-Brand-Guide/Brand-Guide-RU.pdf'
c=canvas.Canvas(str(guide),pagesize=(595.28,841.89))
c.setTitle('Liikkuva — Orange brand kit')
def ptext(x,y,s,size=10,bold=False,color=DARK):
    c.setFillColor(color);c.setFont('DVB' if bold else 'DV',size);c.drawString(x,y,s)
def line(y,s):ptext(40,y,s)
def header(title,n):
    ptext(40,793,title,21,True);ptext(40,766,'liikkuva auto korjaamo  /  Orange edition',10,color='#647069')
    ptext(40,26,'Liikkuva · 11.11.2026',8,color='#647069');ptext(535,26,str(n),8)
header('Логотип и фирменные цвета',1)
y=575
for name,fg,bg in themes:
    c.setFillColor(bg);c.roundRect(40,y,515,153,10,fill=1,stroke=0)
    im=render(docs[name],1600);c.drawImage(ImageReader(im),59,y+18,width=477,height=171.72*.68,mask='auto',preserveAspectRatio=True,anchor='c')
    y-=174
for i,(key,col) in enumerate(palette.items()):
    x=40+i*104;c.setFillColor(col);c.roundRect(x,103,90,35,4,fill=1,stroke=0);ptext(x,85,col,9)
line(57,'На белом фоне — тёмная надпись; на тёмном и зелёном — белая.')
c.showPage()
header('Иконки и применение',2)
for i,(name,fg,bg) in enumerate(themes):
    im=render(icons[name],512);c.drawImage(ImageReader(im),40+i*180,568,width=145,height=145,mask='auto')
    ptext(40+i*180,546,{'white':'Белый фон','dark':'Тёмный фон','green':'Зелёный фон'}[name],11,True)
ptext(40,495,'Проверка малых размеров',14,True)
for i,size in enumerate([16,32,48,64]):
    im=render(icons['dark'],size);x=40+i*130;c.drawImage(ImageReader(im),x,410,width=size*.75,height=size*.75,mask='auto');ptext(x,385,f'{size} px',10)
lines=[
'Основной знак — крупный гаечный ключ. Дополнительный — ключ со стрелкой.',
'Для вкладки браузера: favicon.svg и favicon.ico из папки 07-Web-Ready.',
'Для iPhone/iPad: apple-touch-icon, квадратный непрозрачный файл.',
'Для Android/PWA: icon-192, icon-512 и отдельный maskable-512.',
'Для macOS: файл icon.icns. Для Windows: многоразмерный favicon.ico.',
'Аватар: 1080 × 1080. Квадратный пост: 1080 × 1080.',
'Портрет: 1080 × 1350. История: 1080 × 1920.',
'Горизонтальный макет: 1200 × 630. Презентация: 1920 × 1080.',
'Не растягивайте логотип по одной оси. Сохраняйте соотношение сторон.',
'Оставляйте свободное поле: минимум высота строчной буквы вокруг знака.',
'Полный логотип лучше использовать от 240 px; ниже — отдельную иконку.'
]
for i,t in enumerate(lines):ptext(40,337-i*25,t,9.5)
c.showPage()
header('Какой файл выбрать',3)
rows=[('SVG / SVGZ','Сайт, дизайнер, масштабирование. SVGZ — сжатый SVG.'),('PDF / EPS / PS','Векторные файлы для макетов, документов и печати.'),('EMF / WMF','Вектор для Office и Windows-программ. Предпочтителен EMF.'),('PNG','Прозрачность, документы, мессенджеры, презентации.'),('WebP / AVIF','Современные веб-форматы. AVIF экспортирован с quality 95.'),('JPG','Обычное изображение с готовым фоном; без прозрачности.'),('TIFF','Растровый файл 4000 px с LZW-сжатием и меткой 300 dpi.'),('BMP / GIF','Для старых программ; GIF ограничен палитрой 256 цветов.'),('ICO / ICNS','Значки браузеров, Windows и macOS.')]
y=708
for a,b in rows:ptext(40,y,a,12,True);ptext(40,y-20,b,9.5);y-=50
ptext(40,238,'Перед передачей в типографию',14,True)
for i,t in enumerate([
'Цветные исходники подготовлены в RGB. Это не сертифицированный PDF/X.',
'Для CMYK, плашечного оранжевого или белой краски нужен профиль типографии.',
'Шрифты в логотипах переведены в контуры: устанавливать шрифт не требуется.',
'TIFF 4000 px при 300 dpi соответствует ширине примерно 33,9 см.',
'Для больших вывесок выбирайте SVG, PDF или EPS — они масштабируются.',
'Чёрный и белый одноцветные варианты находятся в папке 03-Print.',
'Это комплект логотипов; вылеты и метки реза задаются в финальном макете.'
]):ptext(40,208-i*21,t,9.3)
c.save()

readme='''НАЧНИТЕ ЗДЕСЬ — LIIKKUVA ORANGE BRAND KIT

Надпись: liikkuva auto korjaamo. Все три слова сохранены.
Логотип взят из предыдущего векторного комплекта: контуры не перерисованы.
Стрелка и маленький акцент окрашены в оранжевый #F97316.
Белый фон: тёмно-синий контур #001A4B. Тёмный фон #17191C и зелёный #123F32: белый контур.

БЫСТРЫЙ ВЫБОР
Для сайта: 07-Web-Ready/logo-transparent-light.svg.
Для тёмного сайта: 07-Web-Ready/logo-transparent-dark.svg.
Для готового белого / тёмного / зелёного фона: logo-white / logo-dark / logo-green.svg.
Для аватарки: 06-Social/<цвет>/avatar-1080x1080.png.
Для favicon: 07-Web-Ready/favicon.svg + favicon.ico.
Для Word, PowerPoint и писем: PNG из 02-Raster.
Для типографии и дизайнера: SVG/PDF/EPS из 01-Vector.
Для печати растрового изображения: TIFF из 03-Print.
Визуальный каталог: откройте INDEX.html. Инструкция: 08-Brand-Guide/Brand-Guide-RU.pdf.

ПАПКИ
01-Vector: SVG, SVGZ, PDF, EPS, PS, EMF, WMF; 3 фона и 2 прозрачных варианта.
02-Raster: PNG и WebP шириной 400, 800, 1600, 3200 px; AVIF 1600 px.
JPG, BMP, GIF имеют готовый фон и не используются как прозрачный логотип.
03-Print: TIFF 4000 px / 300 dpi; одноцветные чёрный и белый варианты.
04-Additional-Lockups: автомобиль без текста и отдельная надпись.
05-Icons: три фона, SVG/PDF/EPS, ICO/ICNS, PNG 16–1024 px, Apple touch, maskable.
06-Social: 6 размеров/назначений для каждого фона, SVG + PNG + JPG.
07-Web-Ready: компактный комплект для подключения к сайту и HTML-пример.
08-Brand-Guide: PDF-инструкция и палитра JSON/CSS/GPL.
09-Rebuild: исходный вектор предыдущего комплекта и скрипт повторного экспорта.

FAVICON
Основная иконка — простой оранжевый гаечный ключ с крупным силуэтом.
Дополнительная — ключ, переходящий в стрелку. Файлы alternative-wrench-arrow.
Обычные иконки имеют прозрачность за скруглёнными углами. Apple touch и maskable — непрозрачные.
Для PWA maskable используйте только отдельный maskable-файл: у него увеличены безопасные поля.
В ICO включены размеры 16/24/32/48/64/128/256. ICNS предназначен для macOS.

ЦВЕТ И ПЕЧАТЬ
Цветные материалы подготовлены в RGB; преобразование в CMYK зависит от профиля типографии.
Файлы не заявлены как PDF/X или готовый макет наклейки с вылетами.
TIFF LZW сохраняет изображение без потерь. 4000 px при 300 dpi — ширина 33,9 см.
SVG/PDF/EPS — настоящие векторы, буквы в контурах. AI/CDR не имитируются переименованием.
AVIF quality 95 — сжатие с потерями; для точного оригинала используйте SVG/PNG/TIFF/WebP lossless.
SVGZ открывается в редакторах; для сайта сервер должен правильно отдавать Content-Encoding gzip.
Белый одноцветный логотип предназначен для тёмной поверхности и может быть не виден на белом листе.
GIF — статический, для старых систем; для красивых краёв выбирайте PNG/WebP/SVG.

СОЦСЕТИ
Файлы содержат логотип на фоне, без рекламного текста и контактных данных.
Размеры универсальные. Кадрирование конкретной платформы проверяйте перед публикацией.

УСТАНОВКА
Скопируйте выбранные файлы 07-Web-Ready в каталог статики вашего сайта.
Используйте относительные пути из example.html или замените их на пути вашей статики.
Существующий site.webmanifest объединяйте с новым, если приложение уже использует PWA.
Этот комплект не публикует изменения на сайте автоматически.
'''
save(Path('START-HERE-RU.txt'),readme)

cards=''.join(f'<article style="background:{bg};color:{fg}"><h2>{name.upper()}</h2>{docs[name]}<p><a href="01-Vector/logo-orange-{name}.svg">SVG</a> · <a href="01-Vector/logo-orange-{name}.pdf">PDF</a> · <a href="02-Raster/{name}/logo-orange-{name}-3200.png">PNG 3200</a></p></article>' for name,fg,bg in themes)
iconcards=''.join(f'<div class="iconcard">{icons[name]}<p>{name}</p><p>'+''.join(f'<img src="05-Icons/{name}/icon-{s}.png" width="{s}" height="{s}" alt="{s} px"> ' for s in [16,32,48,64])+'</p></div>' for name,fg,bg in themes)
catalog='''<!doctype html><html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Liikkuva — Orange Brand Kit</title><style>*{box-sizing:border-box}body{margin:0;background:#eceeeb;color:#17191c;font:16px/1.6 system-ui,sans-serif}main{max-width:1440px;margin:auto;padding:48px 24px}h1{font-size:40px;letter-spacing:-1.5px}.grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:20px}article,.iconcard{padding:28px;border-radius:20px}article>svg{width:100%;height:auto;margin:45px 0}h2{font-size:16px;letter-spacing:2px}a{color:inherit}section{margin-top:48px}.iconcard{background:white}.iconcard>svg{width:128px;height:128px}.iconcard img{vertical-align:middle;margin-right:14px}footer{margin-top:50px;color:#626d67}@media(max-width:850px){.grid{grid-template-columns:1fr}}</style></head><body><main><h1>liikkuva auto korjaamo</h1><p>Orange edition · Векторный логотип, простые иконки и материалы для разных носителей</p><div class="grid">'''+cards+'''</div><section><h1>Простая favicon</h1><p>Крупный гаечный ключ. Ниже показаны реальные размеры 16, 32, 48 и 64 px.</p><div class="grid">'''+iconcards+'''</div></section><section><h1>Файлы и инструкция</h1><p><a href="START-HERE-RU.txt">Начать здесь</a> · <a href="08-Brand-Guide/Brand-Guide-RU.pdf">PDF-инструкция</a> · <a href="07-Web-Ready/example.html">Пример подключения к сайту</a> · <a href="FILES.csv">Каталог всех файлов</a></p><p>SVG · SVGZ · PDF · EPS · PS · PNG · WebP · AVIF · JPG · TIFF · BMP · GIF · ICO · ICNS</p><p>Оранжевый #F97316 · Зелёный #123F32 · Тёмный #17191C</p></section><footer>Полный логотип и надпись сохранены из предыдущего вектора. 11 ноября 2026.</footer></main></body></html>'''
save(Path('INDEX.html'),catalog)
indexpath=OUT/'INDEX.html'
indexpath.write_text(indexpath.read_text().replace('GIF · ICO · ICNS</p>','GIF · ICO · ICNS · EMF · WMF</p>'))
save(Path('09-Rebuild')/'source-master.svg',SOURCE.read_bytes())
save(Path('09-Rebuild')/'build_orange_pack.py',Path(__file__).read_bytes())
save(Path('09-Rebuild')/'requirements.txt','Pillow>=11.3\ncairosvg\nreportlab\npypdf\n')
save(Path('09-Rebuild')/'README.txt','Для повторного экспорта: Python 3, Inkscape, шрифты DejaVu Sans и зависимости requirements.txt. Запустите build_orange_pack.py из отдельной рабочей папки. Скрипт создаёт Liikkuva-Orange-Brand-Kit и ZIP в текущей папке. source-master.svg — исходный синий вектор, используемый только как база для оранжевого комплекта. Для обычного использования эти файлы не нужны.\n')

# Validation: geometry preserved, no remaining blue arrows, decodable files, vector PDFs.
for name,doc in docs.items():
    r=ET.fromstring(doc); pp=[p for p in r.iter() if p.tag.endswith('path')]
    assert len(pp)==len(paths)
    assert [p.attrib['d'] for p in pp]==[p.attrib['d'] for p in paths]
    assert '#0755D8' not in doc
    assert all(p.get('fill') in [ORANGE,WHITE,INK] for p in pp)
for p in OUT.rglob('*'):
    if p.suffix=='.svg':ET.parse(p)
    elif p.suffix in ['.png','.webp','.avif','.jpg','.tif','.bmp','.gif','.ico','.icns']:
        with Image.open(p) as im:im.load()
    elif p.suffix=='.pdf':
        reader=PdfReader(p)
        assert len(reader.pages)>0
        if p!=guide:assert len(reader.pages[0].images)==0
with (OUT/'FILES.csv').open('w',newline='',encoding='utf-8-sig') as f:
    w=csv.writer(f);w.writerow(['File','Format','Bytes','SHA256'])
    for p in sorted(OUT.rglob('*')):
        if p.is_file() and p.name!='FILES.csv':w.writerow([str(p.relative_to(OUT)),p.suffix[1:],p.stat().st_size,hashlib.sha256(p.read_bytes()).hexdigest()])
archive=ROOT/'Liikkuva-Orange-Brand-Kit.zip'
with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED,compresslevel=9) as z:
    for p in sorted(OUT.rglob('*')):
        if p.is_file():z.write(p,Path(OUT.name)/p.relative_to(OUT))
stats=Counter(p.suffix for p in OUT.rglob('*') if p.is_file())
print(json.dumps({'archive':str(archive),'bytes':archive.stat().st_size,'counts':dict(stats),'files':sum(stats.values())},indent=2),flush=True)
