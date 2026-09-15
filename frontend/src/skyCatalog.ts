export type SkyCatalogKind = 'star' | 'messier'

export interface SkyCatalogObject {
  id: string
  name: string
  nameEn: string
  kind: SkyCatalogKind
  raHours: number
  decDegrees: number
  magnitude: number
  constellation: string
}

// J2000 赤经赤纬。星图只保留足以辨认主要星座的亮星，以及适合入门规划的梅西耶精选。
export const skyCatalog: SkyCatalogObject[] = [
  ['sirius','天狼星','SIRIUS','star',6.7525,-16.7161,-1.46,'大犬座'],
  ['canopus','老人星','CANOPUS','star',6.3992,-52.6957,-0.74,'船底座'],
  ['arcturus','大角星','ARCTURUS','star',14.261,19.1825,-0.05,'牧夫座'],
  ['vega','织女星','VEGA','star',18.6156,38.7837,0.03,'天琴座'],
  ['capella','五车二','CAPELLA','star',5.2782,45.998,0.08,'御夫座'],
  ['rigel','参宿七','RIGEL','star',5.2423,-8.2016,0.13,'猎户座'],
  ['procyon','南河三','PROCYON','star',7.655,5.225,0.34,'小犬座'],
  ['betelgeuse','参宿四','BETELGEUSE','star',5.9195,7.4071,0.42,'猎户座'],
  ['achernar','水委一','ACHERNAR','star',1.6286,-57.2368,0.46,'波江座'],
  ['altair','河鼓二','ALTAIR','star',19.8464,8.8683,0.77,'天鹰座'],
  ['aldebaran','毕宿五','ALDEBARAN','star',4.5987,16.5093,0.85,'金牛座'],
  ['antares','心宿二','ANTARES','star',16.4901,-26.432,0.96,'天蝎座'],
  ['spica','角宿一','SPICA','star',13.4199,-11.1613,0.97,'室女座'],
  ['pollux','北河三','POLLUX','star',7.7553,28.0262,1.14,'双子座'],
  ['fomalhaut','北落师门','FOMALHAUT','star',22.9608,-29.6222,1.16,'南鱼座'],
  ['deneb','天津四','DENEB','star',20.6905,45.2803,1.25,'天鹅座'],
  ['regulus','轩辕十四','REGULUS','star',10.1395,11.9672,1.35,'狮子座'],
  ['castor','北河二','CASTOR','star',7.5767,31.8883,1.58,'双子座'],
  ['bellatrix','参宿五','BELLATRIX','star',5.4189,6.3497,1.64,'猎户座'],
  ['alnilam','参宿二','ALNILAM','star',5.6036,-1.2019,1.69,'猎户座'],
  ['alnitak','参宿一','ALNITAK','star',5.6793,-1.9426,1.74,'猎户座'],
  ['alioth','玉衡','ALIOTH','star',12.9005,55.9598,1.76,'大熊座'],
  ['dubhe','天枢','DUBHE','star',11.0621,61.7508,1.79,'大熊座'],
  ['mirfak','天船三','MIRFAK','star',3.4054,49.8612,1.79,'英仙座'],
  ['alkaid','摇光','ALKAID','star',13.7923,49.3133,1.86,'大熊座'],
  ['mizar','开阳','MIZAR','star',13.3987,54.9254,2.23,'大熊座'],
  ['merak','天璇','MERAK','star',11.0307,56.3824,2.37,'大熊座'],
  ['phecda','天玑','PHECDA','star',11.8972,53.6948,2.44,'大熊座'],
  ['megrez','天权','MEGREZ','star',12.257,57.0326,3.31,'大熊座'],
  ['schedar','王良四','SCHEDAR','star',0.6751,56.5373,2.24,'仙后座'],
  ['caph','王良一','CAPH','star',0.1529,59.1498,2.27,'仙后座'],
  ['gamma-cas','策','GAMMA CAS','star',0.9451,60.7167,2.47,'仙后座'],
  ['ruchbah','阁道三','RUCHBAH','star',1.4303,60.2353,2.68,'仙后座'],
  ['segin','阁道二','SEGIN','star',1.9066,63.67,3.35,'仙后座'],
  ['saiph','参宿六','SAIPH','star',5.7959,-9.6696,2.06,'猎户座'],
  ['mintaka','参宿三','MINTAKA','star',5.5334,-0.2991,2.23,'猎户座'],
  ['m1','蟹状星云','M1','messier',5.5756,22.0145,8.4,'金牛座'],
  ['m8','礁湖星云','M8','messier',18.0603,-24.3867,6.0,'人马座'],
  ['m13','武仙座大星团','M13','messier',16.6949,36.4613,5.8,'武仙座'],
  ['m20','三叶星云','M20','messier',18.0414,-23.0297,6.3,'人马座'],
  ['m27','哑铃星云','M27','messier',19.9934,22.7212,7.5,'狐狸座'],
  ['m31','仙女座星系','M31','messier',0.7123,41.2692,3.4,'仙女座'],
  ['m42','猎户座大星云','M42','messier',5.5881,-5.3911,4.0,'猎户座'],
  ['m44','蜂巢星团','M44','messier',8.6728,19.6721,3.7,'巨蟹座'],
  ['m45','昴星团','M45','messier',3.79,24.1167,1.6,'金牛座'],
  ['m51','涡状星系','M51','messier',13.4979,47.1952,8.4,'猎犬座'],
  ['m57','环状星云','M57','messier',18.8931,33.0292,8.8,'天琴座'],
  ['m81','波德星系','M81','messier',9.9259,69.0653,6.9,'大熊座'],
  ['m82','雪茄星系','M82','messier',9.9313,69.6797,8.4,'大熊座'],
  ['m101','风车星系','M101','messier',14.0535,54.3488,7.9,'大熊座'],
  ['m104','草帽星系','M104','messier',12.6665,-11.6231,8.0,'室女座'],
].map(([id,name,nameEn,kind,raHours,decDegrees,magnitude,constellation]) => ({
  id: String(id), name: String(name), nameEn: String(nameEn), kind: kind as SkyCatalogKind,
  raHours: Number(raHours), decDegrees: Number(decDegrees), magnitude: Number(magnitude), constellation: String(constellation),
}))

export const constellationLines = [
  { name: '猎户座', segments: [['betelgeuse','bellatrix'],['betelgeuse','alnitak'],['bellatrix','mintaka'],['alnitak','alnilam'],['alnilam','mintaka'],['alnitak','saiph'],['mintaka','rigel'],['saiph','rigel']] },
  { name: '大熊座', segments: [['dubhe','merak'],['merak','phecda'],['phecda','megrez'],['megrez','dubhe'],['megrez','alioth'],['alioth','mizar'],['mizar','alkaid']] },
  { name: '仙后座', segments: [['caph','schedar'],['schedar','gamma-cas'],['gamma-cas','ruchbah'],['ruchbah','segin']] },
].map((item) => ({ ...item, segments: item.segments as Array<[string,string]> }))

// 银河中心带的粗略 J2000 中心线，只表达方向与跨度，不冒充精密全天巡天图。
export const milkyWayCenterline = [
  [0.8,62],[3.2,52],[5.8,20],[7.2,-10],[9.5,-45],[12.5,-62],[15.5,-48],[17.8,-30],[18.7,0],[20.3,35],[22.4,55],[24.8,62],
].map(([raHours, decDegrees]) => ({ raHours, decDegrees }))
