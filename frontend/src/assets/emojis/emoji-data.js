// 本地化 Emoji 数据
// 使用本地 SVG 图标，确保离线环境可用

// Emoji 分类
export const EMOJI_CATEGORIES = [
  { key: 'frequent', name: '常用', icon: '🕒' },
  { key: 'smileys', name: '笑脸', icon: '😀' },
  { key: 'gestures', name: '手势', icon: '👋' },
  { key: 'people', name: '人物', icon: '👨' },
  { key: 'animals', name: '动物', icon: '🐶' },
  { key: 'food', name: '食物', icon: '🍕' },
  { key: 'activities', name: '活动', icon: '⚽' },
  { key: 'travel', name: '旅行', icon: '✈️' },
  { key: 'objects', name: '物品', icon: '💡' },
  { key: 'symbols', name: '符号', icon: '❤️' },
  { key: 'flags', name: '旗帜', icon: '🚩' }
]

// 基础 Emoji 集合（使用 Unicode，但提供本地 SVG 备用）
export const EMOJI_DATA = {
  smileys: [
    { code: '1f600', char: '😀', name: '笑脸', keywords: ['笑', 'smile', 'happy'] },
    { code: '1f603', char: '😃', name: '大笑', keywords: ['笑', 'laugh', 'happy'] },
    { code: '1f604', char: '😄', name: '开心', keywords: ['笑', 'happy', 'joy'] },
    { code: '1f601', char: '😁', name: '露齿笑', keywords: ['笑', 'grin', 'happy'] },
    { code: '1f606', char: '😆', name: '哈哈', keywords: ['笑', 'laugh', 'haha'] },
    { code: '1f605', char: '😅', name: '汗笑', keywords: ['笑', 'sweat', 'nervous'] },
    { code: '1f923', char: '🤣', name: '笑哭', keywords: ['笑', 'rofl', 'laugh'] },
    { code: '1f602', char: '😂', name: '喜极而泣', keywords: ['笑', 'cry', 'tears'] },
    { code: '1f642', char: '🙂', name: '微笑', keywords: ['笑', 'smile', 'slight'] },
    { code: '1f643', char: '🙃', name: '倒脸', keywords: ['笑', 'upside', 'down'] },
    { code: '1f609', char: '😉', name: '眨眼', keywords: ['wink', '眨眼'] },
    { code: '1f60a', char: '😊', name: '羞涩', keywords: ['blush', '害羞'] },
    { code: '1f607', char: '😇', name: '天使', keywords: ['angel', '天使'] },
    { code: '1f970', char: '🥰', name: '爱心脸', keywords: ['love', '爱'] },
    { code: '1f60d', char: '😍', name: '爱慕', keywords: ['love', '爱'] },
    { code: '1f929', char: '🤩', name: '崇拜', keywords: ['star', '明星'] },
    { code: '1f618', char: '😘', name: '飞吻', keywords: ['kiss', '吻'] },
    { code: '1f617', char: '😗', name: '接吻', keywords: ['kiss', '吻'] },
    { code: '1f61a', char: '😚', name: '闭眼吻', keywords: ['kiss', '吻'] },
    { code: '1f619', char: '😙', name: '微笑吻', keywords: ['kiss', '吻'] },
    { code: '1f60b', char: '😋', name: '好吃', keywords: ['yum', '好吃'] },
    { code: '1f61b', char: '😛', name: '吐舌', keywords: ['tongue', '舌头'] },
    { code: '1f61c', char: '😜', name: '眨眼吐舌', keywords: ['wink', 'tongue'] },
    { code: '1f92a', char: '🤪', name: '疯狂', keywords: ['crazy', '疯狂'] },
    { code: '1f61d', char: '😝', name: '闭眼吐舌', keywords: ['tongue', '舌头'] },
    { code: '1f911', char: '🤑', name: '发财', keywords: ['money', '钱'] },
    { code: '1f917', char: '🤗', name: '拥抱', keywords: ['hug', '拥抱'] },
    { code: '1f92d', char: '🤭', name: '捂嘴笑', keywords: ['giggle', '笑'] },
    { code: '1f92b', char: '🤫', name: '嘘', keywords: ['shh', '安静'] },
    { code: '1f914', char: '🤔', name: '思考', keywords: ['think', '思考'] },
    { code: '1f910', char: '🤐', name: '闭嘴', keywords: ['zipper', '拉链'] },
    { code: '1f928', char: '🤨', name: '质疑', keywords: ['raise', 'eyebrow'] },
    { code: '1f610', char: '😐', name: '面无表情', keywords: ['neutral', '中性'] },
    { code: '1f611', char: '😑', name: '无语', keywords: ['expressionless', '无语'] },
    { code: '1f636', char: '😶', name: '沉默', keywords: ['silent', '沉默'] },
    { code: '1f60f', char: '😏', name: '得意', keywords: ['smirk', '得意'] },
    { code: '1f612', char: '😒', name: '不悦', keywords: ['unamused', '不爽'] },
    { code: '1f644', char: '🙄', name: '翻白眼', keywords: ['roll', 'eyes'] },
    { code: '1f62c', char: '😬', name: '尴尬', keywords: ['grimace', '尴尬'] },
    { code: '1f925', char: '🤥', name: '撒谎', keywords: ['lie', '撒谎'] },
    { code: '1f60c', char: '😌', name: '释然', keywords: ['relieved', '放心'] },
    { code: '1f614', char: '😔', name: '沉思', keywords: ['pensive', '沉思'] },
    { code: '1f62a', char: '😪', name: '困倦', keywords: ['sleepy', '困'] },
    { code: '1f924', char: '🤤', name: '流口水', keywords: ['drool', '口水'] },
    { code: '1f634', char: '😴', name: '睡觉', keywords: ['sleep', '睡觉'] },
    { code: '1f637', char: '😷', name: '口罩', keywords: ['mask', '口罩'] },
    { code: '1f912', char: '🤒', name: '生病', keywords: ['sick', '生病'] },
    { code: '1f915', char: '🤕', name: '受伤', keywords: ['injured', '受伤'] },
    { code: '1f922', char: '🤢', name: '恶心', keywords: ['nausea', '恶心'] },
    { code: '1f92e', char: '🤮', name: '呕吐', keywords: ['vomit', '吐'] },
    { code: '1f927', char: '🤧', name: '打喷嚏', keywords: ['sneeze', '喷嚏'] },
    { code: '1f975', char: '🥵', name: '热', keywords: ['hot', '热'] },
    { code: '1f976', char: '🥶', name: '冷', keywords: ['cold', '冷'] },
    { code: '1f635', char: '😵', name: '晕', keywords: ['dizzy', '晕'] },
    { code: '1f92f', char: '🤯', name: '爆炸头', keywords: ['explode', '爆炸'] },
    { code: '1f920', char: '🤠', name: '牛仔', keywords: ['cowboy', '牛仔'] },
    { code: '1f973', char: '🥳', name: '派对', keywords: ['party', '派对'] },
    { code: '1f60e', char: '😎', name: '墨镜', keywords: ['cool', '酷'] },
    { code: '1f913', char: '🤓', name: '书呆子', keywords: ['nerd', '书呆子'] },
    { code: '1f9d0', char: '🧐', name: '单片眼镜', keywords: ['monocle', '眼镜'] }
  ],
  gestures: [
    { code: '1f44b', char: '👋', name: '挥手', keywords: ['wave', '挥手', 'hello'] },
    { code: '1f91a', char: '🤚', name: '举手', keywords: ['raised', '举手'] },
    { code: '1f590', char: '🖐️', name: '张开手', keywords: ['hand', '手'] },
    { code: '270b', char: '✋', name: '停', keywords: ['stop', '停止'] },
    { code: '1f596', char: '🖖', name: '瓦肯礼', keywords: ['vulcan', 'star trek'] },
    { code: '1f44c', char: '👌', name: 'OK', keywords: ['ok', 'okay'] },
    { code: '1f90c', char: '🤌', name: '捏手指', keywords: ['pinch', '捏'] },
    { code: '1f90f', char: '🤏', name: '一点点', keywords: ['pinch', '一点'] },
    { code: '270c', char: '✌️', name: 'V', keywords: ['victory', '胜利'] },
    { code: '1f91e', char: '🤞', name: '交叉手指', keywords: ['cross', '祈祷'] },
    { code: '1f91f', char: '🤟', name: '爱你', keywords: ['love', '爱'] },
    { code: '1f918', char: '🤘', name: '摇滚', keywords: ['rock', '摇滚'] },
    { code: '1f919', char: '🤙', name: '打电话', keywords: ['call', '电话'] },
    { code: '1f448', char: '👈', name: '左指', keywords: ['left', '左'] },
    { code: '1f449', char: '👉', name: '右指', keywords: ['right', '右'] },
    { code: '1f446', char: '👆', name: '上指', keywords: ['up', '上'] },
    { code: '1f447', char: '👇', name: '下指', keywords: ['down', '下'] },
    { code: '261d', char: '☝️', name: '食指', keywords: ['index', '指'] },
    { code: '1f44d', char: '👍', name: '点赞', keywords: ['thumbs', 'up', '赞'] },
    { code: '1f44e', char: '👎', name: '点踩', keywords: ['thumbs', 'down', '踩'] },
    { code: '270a', char: '✊', name: '拳头', keywords: ['fist', '拳'] },
    { code: '1f44a', char: '👊', name: '对拳', keywords: ['punch', '拳'] },
    { code: '1f91b', char: '🤛', name: '左拳', keywords: ['left', 'punch'] },
    { code: '1f91c', char: '🤜', name: '右拳', keywords: ['right', 'punch'] },
    { code: '1f44f', char: '👏', name: '鼓掌', keywords: ['clap', '鼓掌'] },
    { code: '1f64c', char: '🙌', name: '举双手', keywords: ['raise', '举手'] },
    { code: '1f450', char: '👐', name: '张开双手', keywords: ['open', 'hands'] },
    { code: '1f932', char: '🤲', name: '捧', keywords: ['palms', '手掌'] },
    { code: '1f91d', char: '🤝', name: '握手', keywords: ['handshake', '握手'] },
    { code: '1f64f', char: '🙏', name: '祈祷', keywords: ['pray', '祈祷', 'thanks'] }
  ],
  animals: [
    { code: '1f436', char: '🐶', name: '狗', keywords: ['dog', '狗'] },
    { code: '1f431', char: '🐱', name: '猫', keywords: ['cat', '猫'] },
    { code: '1f42d', char: '🐭', name: '鼠', keywords: ['mouse', '老鼠'] },
    { code: '1f439', char: '🐹', name: '仓鼠', keywords: ['hamster', '仓鼠'] },
    { code: '1f430', char: '🐰', name: '兔子', keywords: ['rabbit', '兔子'] },
    { code: '1f98a', char: '🦊', name: '狐狸', keywords: ['fox', '狐狸'] },
    { code: '1f43b', char: '🐻', name: '熊', keywords: ['bear', '熊'] },
    { code: '1f43c', char: '🐼', name: '熊猫', keywords: ['panda', '熊猫'] },
    { code: '1f428', char: '🐨', name: '考拉', keywords: ['koala', '考拉'] },
    { code: '1f42f', char: '🐯', name: '老虎', keywords: ['tiger', '老虎'] },
    { code: '1f981', char: '🦁', name: '狮子', keywords: ['lion', '狮子'] },
    { code: '1f42e', char: '🐮', name: '牛', keywords: ['cow', '牛'] },
    { code: '1f437', char: '🐷', name: '猪', keywords: ['pig', '猪'] },
    { code: '1f438', char: '🐸', name: '青蛙', keywords: ['frog', '青蛙'] },
    { code: '1f435', char: '🐵', name: '猴子', keywords: ['monkey', '猴子'] },
    { code: '1f414', char: '🐔', name: '鸡', keywords: ['chicken', '鸡'] },
    { code: '1f427', char: '🐧', name: '企鹅', keywords: ['penguin', '企鹅'] },
    { code: '1f426', char: '🐦', name: '鸟', keywords: ['bird', '鸟'] },
    { code: '1f424', char: '🐤', name: '小鸡', keywords: ['chick', '小鸡'] },
    { code: '1f986', char: '🦆', name: '鸭', keywords: ['duck', '鸭子'] }
  ],
  food: [
    { code: '1f34e', char: '🍎', name: '苹果', keywords: ['apple', '苹果'] },
    { code: '1f34a', char: '🍊', name: '橘子', keywords: ['orange', '橘子'] },
    { code: '1f34b', char: '🍋', name: '柠檬', keywords: ['lemon', '柠檬'] },
    { code: '1f34c', char: '🍌', name: '香蕉', keywords: ['banana', '香蕉'] },
    { code: '1f349', char: '🍉', name: '西瓜', keywords: ['watermelon', '西瓜'] },
    { code: '1f347', char: '🍇', name: '葡萄', keywords: ['grapes', '葡萄'] },
    { code: '1f353', char: '🍓', name: '草莓', keywords: ['strawberry', '草莓'] },
    { code: '1f351', char: '🍑', name: '桃子', keywords: ['peach', '桃子'] },
    { code: '1f352', char: '🍒', name: '樱桃', keywords: ['cherry', '樱桃'] },
    { code: '1f345', char: '🍅', name: '番茄', keywords: ['tomato', '番茄'] },
    { code: '1f35e', char: '🍞', name: '面包', keywords: ['bread', '面包'] },
    { code: '1f9c0', char: '🧀', name: '奶酪', keywords: ['cheese', '奶酪'] },
    { code: '1f356', char: '🍖', name: '肉', keywords: ['meat', '肉'] },
    { code: '1f357', char: '🍗', name: '鸡腿', keywords: ['chicken', '鸡腿'] },
    { code: '1f354', char: '🍔', name: '汉堡', keywords: ['burger', '汉堡'] },
    { code: '1f35f', char: '🍟', name: '薯条', keywords: ['fries', '薯条'] },
    { code: '1f355', char: '🍕', name: '披萨', keywords: ['pizza', '披萨'] },
    { code: '1f32d', char: '🌭', name: '热狗', keywords: ['hotdog', '热狗'] },
    { code: '1f96a', char: '🥪', name: '三明治', keywords: ['sandwich', '三明治'] },
    { code: '1f373', char: '🍳', name: '煎蛋', keywords: ['egg', '鸡蛋'] }
  ],
  activities: [
    { code: '26bd', char: '⚽', name: '足球', keywords: ['soccer', '足球'] },
    { code: '1f3c0', char: '🏀', name: '篮球', keywords: ['basketball', '篮球'] },
    { code: '1f3c8', char: '🏈', name: '橄榄球', keywords: ['football', '橄榄球'] },
    { code: '26be', char: '⚾', name: '棒球', keywords: ['baseball', '棒球'] },
    { code: '1f3be', char: '🎾', name: '网球', keywords: ['tennis', '网球'] },
    { code: '1f3d0', char: '🏐', name: '排球', keywords: ['volleyball', '排球'] },
    { code: '1f3d3', char: '🏓', name: '乒乓球', keywords: ['ping pong', '乒乓球'] },
    { code: '1f3f8', char: '🏸', name: '羽毛球', keywords: ['badminton', '羽毛球'] },
    { code: '1f945', char: '🥅', name: '球门', keywords: ['goal', '球门'] },
    { code: '1f3af', char: '🎯', name: '靶心', keywords: ['target', '靶心'] },
    { code: '1f3ae', char: '🎮', name: '游戏手柄', keywords: ['game', '游戏'] },
    { code: '1f579', char: '🕹️', name: '操纵杆', keywords: ['joystick', '摇杆'] },
    { code: '1f3b2', char: '🎲', name: '骰子', keywords: ['dice', '骰子'] },
    { code: '1f3ad', char: '🎭', name: '面具', keywords: ['mask', '面具'] },
    { code: '1f3a8', char: '🎨', name: '调色板', keywords: ['art', '艺术'] },
    { code: '1f3ac', char: '🎬', name: '场记板', keywords: ['movie', '电影'] },
    { code: '1f3a4', char: '🎤', name: '麦克风', keywords: ['microphone', '麦克风'] },
    { code: '1f3a7', char: '🎧', name: '耳机', keywords: ['headphone', '耳机'] },
    { code: '1f3b8', char: '🎸', name: '吉他', keywords: ['guitar', '吉他'] },
    { code: '1f3b9', char: '🎹', name: '钢琴', keywords: ['piano', '钢琴'] }
  ],
  travel: [
    { code: '1f697', char: '🚗', name: '汽车', keywords: ['car', '汽车'] },
    { code: '1f695', char: '🚕', name: '出租车', keywords: ['taxi', '出租车'] },
    { code: '1f699', char: '🚙', name: 'SUV', keywords: ['suv', '越野车'] },
    { code: '1f68c', char: '🚌', name: '公交车', keywords: ['bus', '公交车'] },
    { code: '1f68e', char: '🚎', name: '无轨电车', keywords: ['trolley', '电车'] },
    { code: '1f3ce', char: '🏎️', name: '赛车', keywords: ['race', '赛车'] },
    { code: '1f693', char: '🚓', name: '警车', keywords: ['police', '警车'] },
    { code: '1f691', char: '🚑', name: '救护车', keywords: ['ambulance', '救护车'] },
    { code: '1f692', char: '🚒', name: '消防车', keywords: ['fire', '消防车'] },
    { code: '1f69a', char: '🚚', name: '卡车', keywords: ['truck', '卡车'] },
    { code: '1f6b2', char: '🚲', name: '自行车', keywords: ['bike', '自行车'] },
    { code: '1f3cd', char: '🏍️', name: '摩托车', keywords: ['motorcycle', '摩托车'] },
    { code: '2708', char: '✈️', name: '飞机', keywords: ['plane', '飞机'] },
    { code: '1f680', char: '🚀', name: '火箭', keywords: ['rocket', '火箭'] },
    { code: '1f6f8', char: '🛸', name: 'UFO', keywords: ['ufo', 'alien'] },
    { code: '1f681', char: '🚁', name: '直升机', keywords: ['helicopter', '直升机'] },
    { code: '26f5', char: '⛵', name: '帆船', keywords: ['sailboat', '帆船'] },
    { code: '1f6a4', char: '🚤', name: '快艇', keywords: ['speedboat', '快艇'] },
    { code: '1f6a2', char: '🚢', name: '轮船', keywords: ['ship', '轮船'] },
    { code: '2693', char: '⚓', name: '锚', keywords: ['anchor', '锚'] }
  ],
  objects: [
    { code: '1f4f1', char: '📱', name: '手机', keywords: ['phone', '手机'] },
    { code: '1f4bb', char: '💻', name: '笔记本', keywords: ['laptop', '笔记本'] },
    { code: '2328', char: '⌨️', name: '键盘', keywords: ['keyboard', '键盘'] },
    { code: '1f5a5', char: '🖥️', name: '台式机', keywords: ['desktop', '台式机'] },
    { code: '1f5a8', char: '🖨️', name: '打印机', keywords: ['printer', '打印机'] },
    { code: '1f5b1', char: '🖱️', name: '鼠标', keywords: ['mouse', '鼠标'] },
    { code: '1f4be', char: '💾', name: '软盘', keywords: ['floppy', '软盘'] },
    { code: '1f4bf', char: '💿', name: 'CD', keywords: ['cd', 'disc'] },
    { code: '1f4c0', char: '📀', name: 'DVD', keywords: ['dvd', 'disc'] },
    { code: '1f4f7', char: '📷', name: '相机', keywords: ['camera', '相机'] },
    { code: '1f4a1', char: '💡', name: '灯泡', keywords: ['bulb', '灯泡', 'idea'] },
    { code: '1f526', char: '🔦', name: '手电筒', keywords: ['flashlight', '手电筒'] },
    { code: '1f50b', char: '🔋', name: '电池', keywords: ['battery', '电池'] },
    { code: '1f50c', char: '🔌', name: '插头', keywords: ['plug', '插头'] },
    { code: '1f4e1', char: '📡', name: '卫星天线', keywords: ['satellite', '卫星'] },
    { code: '1f512', char: '🔒', name: '锁', keywords: ['lock', '锁'] },
    { code: '1f513', char: '🔓', name: '解锁', keywords: ['unlock', '解锁'] },
    { code: '1f511', char: '🔑', name: '钥匙', keywords: ['key', '钥匙'] },
    { code: '1f528', char: '🔨', name: '锤子', keywords: ['hammer', '锤子'] },
    { code: '1f527', char: '🔧', name: '扳手', keywords: ['wrench', '扳手'] }
  ],
  symbols: [
    { code: '2764', char: '❤️', name: '红心', keywords: ['heart', '爱', 'love'] },
    { code: '1f9e1', char: '🧡', name: '橙心', keywords: ['heart', 'orange'] },
    { code: '1f49b', char: '💛', name: '黄心', keywords: ['heart', 'yellow'] },
    { code: '1f49a', char: '💚', name: '绿心', keywords: ['heart', 'green'] },
    { code: '1f499', char: '💙', name: '蓝心', keywords: ['heart', 'blue'] },
    { code: '1f49c', char: '💜', name: '紫心', keywords: ['heart', 'purple'] },
    { code: '1f5a4', char: '🖤', name: '黑心', keywords: ['heart', 'black'] },
    { code: '1f90d', char: '🤍', name: '白心', keywords: ['heart', 'white'] },
    { code: '1f494', char: '💔', name: '心碎', keywords: ['broken', 'heart'] },
    { code: '2b50', char: '⭐', name: '星星', keywords: ['star', '星星'] },
    { code: '1f31f', char: '🌟', name: '闪光星', keywords: ['star', 'glow'] },
    { code: '2728', char: '✨', name: '闪烁', keywords: ['sparkle', '闪'] },
    { code: '26a1', char: '⚡', name: '闪电', keywords: ['lightning', '闪电'] },
    { code: '1f4a5', char: '💥', name: '碰撞', keywords: ['boom', '爆炸'] },
    { code: '1f4ab', char: '💫', name: '晕眩', keywords: ['dizzy', '晕'] },
    { code: '1f4a6', char: '💦', name: '汗滴', keywords: ['sweat', '汗'] },
    { code: '1f4a8', char: '💨', name: '冲刺', keywords: ['dash', '冲'] },
    { code: '1f525', char: '🔥', name: '火', keywords: ['fire', '火'] },
    { code: '1f4af', char: '💯', name: '一百', keywords: ['100', '满分'] },
    { code: '2705', char: '✅', name: '勾选', keywords: ['check', '对'] },
    { code: '274c', char: '❌', name: '叉', keywords: ['x', '错'] },
    { code: '2757', char: '❗', name: '感叹号', keywords: ['exclamation', '感叹'] },
    { code: '2753', char: '❓', name: '问号', keywords: ['question', '问'] },
    { code: '26a0', char: '⚠️', name: '警告', keywords: ['warning', '警告'] },
    { code: '1f6ab', char: '🚫', name: '禁止', keywords: ['no', '禁止'] }
  ],
  flags: [
    { code: '1f3c1', char: '🏁', name: '赛车旗', keywords: ['race', 'flag'] },
    { code: '1f6a9', char: '🚩', name: '三角旗', keywords: ['flag', '旗'] },
    { code: '1f3f4', char: '🏴', name: '黑旗', keywords: ['black', 'flag'] },
    { code: '1f3f3', char: '🏳️', name: '白旗', keywords: ['white', 'flag'] },
    { code: '1f1e8-1f1f3', char: '🇨🇳', name: '中国', keywords: ['china', '中国'] },
    { code: '1f1fa-1f1f8', char: '🇺🇸', name: '美国', keywords: ['usa', '美国'] },
    { code: '1f1ec-1f1e7', char: '🇬🇧', name: '英国', keywords: ['uk', '英国'] },
    { code: '1f1ef-1f1f5', char: '🇯🇵', name: '日本', keywords: ['japan', '日本'] },
    { code: '1f1f0-1f1f7', char: '🇰🇷', name: '韩国', keywords: ['korea', '韩国'] },
    { code: '1f1eb-1f1f7', char: '🇫🇷', name: '法国', keywords: ['france', '法国'] },
    { code: '1f1e9-1f1ea', char: '🇩🇪', name: '德国', keywords: ['germany', '德国'] },
    { code: '1f1ee-1f1f9', char: '🇮🇹', name: '意大利', keywords: ['italy', '意大利'] },
    { code: '1f1ea-1f1f8', char: '🇪🇸', name: '西班牙', keywords: ['spain', '西班牙'] },
    { code: '1f1f7-1f1fa', char: '🇷🇺', name: '俄罗斯', keywords: ['russia', '俄罗斯'] },
    { code: '1f1e8-1f1e6', char: '🇨🇦', name: '加拿大', keywords: ['canada', '加拿大'] }
  ]
}

// 生成 SVG 备用方案的路径
export const getEmojiAssetPath = (code) => {
  return `/src/assets/emojis/svg/${code}.svg`
}

// 搜索 Emoji
export function searchEmojis(query) {
  if (!query) return []
  
  const lowerQuery = query.toLowerCase()
  const results = []
  
  Object.values(EMOJI_DATA).forEach(category => {
    category.forEach(emoji => {
      if (
        emoji.name.toLowerCase().includes(lowerQuery) ||
        emoji.keywords.some(k => k.toLowerCase().includes(lowerQuery)) ||
        emoji.char.includes(query)
      ) {
        results.push(emoji)
      }
    })
  })
  
  return results.slice(0, 50) // 限制结果数量
}

// 获取所有 Emoji
export function getAllEmojis() {
  return EMOJI_DATA
}

// 根据分类获取 Emoji
export function getEmojisByCategory(category) {
  return EMOJI_DATA[category] || []
}
