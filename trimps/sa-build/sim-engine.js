/* Spire Assault Combat Simulation Engine
 * Derived directly from Trimps Spire Assault combat mechanics (trimps/objects.js)
 * Guarantees 100% mathematical parity with in-game battle execution.
 */
(function(root) {
  var autoBattle = null;
  var sharedEnemyTemplateCache = {};
  // Seeded Random Number Generator (from Trimps main.js)
function seededRandom(seed){
	var x = Math.sin(seed++) * 10000;
	return parseFloat((x - Math.floor(x)).toFixed(7));
}

function getRandomIntSeeded(seed, minIncl, maxExcl) {
	var toReturn = Math.floor(seededRandom(seed) * (maxExcl - minIncl)) + minIncl;
	return (toReturn === maxExcl) ? minIncl : toReturn;
}

  function createSimInstance() {
    var gameMock = {
      badGuys: {
        Tauntimp: { expandingBase: function() { return 0; } }
      },
      portal: {
        Championism: { radLocked: false },
        Expansion: { radLocked: false }
      },
      options: {
        menu: { masteryTab: { enabled: 0 } }
      },
      resources: {
        science: { owned: 0 }
      },
      global: {
        highestRadonLevelCleared: 205,
        fluffyExp2: 0,
        u2MutationData: {},
        autoBattleData: {}
      },
      stats: {
        saKills: { value: 0 },
        saDust: { value: 0 },
        saShards: { value: 0 },
        saHighestLevel: { valueTotal: 71 }
      }
    };

    var u2MutationsMock = {
      tree: {
        Dust: { purchased: false },
        Dust2: { purchased: false }
      }
    };

    function prettify(val) {
      if (!isFinite(val)) return "0";
      if (val >= 1e12) return (val / 1e12).toFixed(2) + "T";
      if (val >= 1e9) return (val / 1e9).toFixed(2) + "B";
      if (val >= 1e6) return (val / 1e6).toFixed(2) + "M";
      if (val >= 1e3) return (val / 1e3).toFixed(1) + "K";
      return Math.round(val).toString();
    }

    var document = {
      getElementById: function() { return { style: {} }; },
      querySelectorAll: function() { return []; }
    };
    function giveSingleAchieve() {}
    function swapClass() {}
    function cancelTooltip() {}
    function tooltip() {}
    var usingRealTimeOffline = false;

function createRawAutoBattle(game, u2Mutations) {
  var autoBattle = {
    frameTime: 300,
    speed: 1,
    enemyLevel: 1,
    maxEnemyLevel: 1,
    autoLevel: true,
    dust: 0,
    shards: 0,
    shardDust: 0,
    trimp: null,
    enemy: null,
    seed: 4568654,
    enemiesKilled: 0,
    sessionEnemiesKilled: 0,
    sessionTrimpsKilled: 0,
    maxItems: 4,
    notes: "&nbsp;",
    popupMode: "items",
    battleTime: 0,
    lastSelect: "",
    lastActions: [],
    activeContract: "",
    sealed: false,
    canSeal: false,
    lootAvg: {
        accumulator: 0,
        counter: 0
    },
    presets: {
        names: ["Preset 1", "Preset 2", "Preset 3"],
        p1: [],
        p2: [],
        p3: []
    },
    rings: {
        level: 1,
        mods: ["attack"]
    },
    template: function(){
        return {
            level: 1,
            isTrimp: false,
            baseHealth: 50,
            health: 50,
            maxHealth: 50,
            baseAttack: 5,
            attack: 5,
            baseAttackSpeed: 5000,
            attackSpeed: 5000,
            lastAttack: 0,
            shockChance: 0,
            shockMod: 0,
            bleedChance: 0,
            bleedMod: 0,
            bleedTime: 0,
            hadBleed: false,
            poisonChance: 0,
            poisonTime: 0,
            poisonMod: 0,
            poisonStack: 2,
            poisonRate: 1,
            poisonTick: 1000,
            poisonHeal: 0,
            defense: 0,
            lifesteal: 0,
            shockResist: 0,
            poisonResist: 0,
            bleedResist: 0,
            lifestealResist: 0,
            slowAura: 1,
            damageTakenMult: 1,
            enrageMult: 1.25,
            enrageFreq: 60,
            explodeDamage: 0,
            explodeFreq: -1,
            lastExplode: 0,
            berserkMod: -1,
            berserkStack: 0,
            ethChance: 0,
            dmgTaken: 0,
            dustMult: 0,
            gooStored: 0,
            lastGoo: -1,
            immune: "",
            bleed: {
                time: 0,
                mod: 0
            },
            poison: {
                time: 0,
                mod: 0,
                lastTick: 0,
                stacks: 0,
                expired: false,
                hitsAtMax: 0
            },
            shock: {
                time: 0,
                mod: 0,
                count: 0,
                timeApplied: 0
            }
        }
    },
    unlockAllItems: function(){
        for (var item in this.items){
            this.items[item].owned = true;
        }
    },

    resetAll: function(){
        this.enemyLevel = 1;
        this.maxEnemyLevel = 1;
        this.autoLevel = true;
        this.dust = 0;
        this.shards = 0;
        this.trimp = null;
        this.enemy = null;
        this.enemiesKilled = 0;
        this.lastActions = [];
        this.activeContract = "";
        this.resetStats();
        this.rings = this.getFreshRings();
        this.sealed = false;
        this.canSeal = false;
        for (var item in this.items){
            item = this.items[item];
            item.owned = (item.zone) ? false : true;
            item.equipped = false;
            item.hidden = false;
            item.level = 1;
        }
        for (var bonus in this.bonuses){
            this.bonuses[bonus].level = 0;
        }
        for (var oneTimer in this.oneTimers){
            this.oneTimers[oneTimer].owned = false;
        }
        for (var setting in this.settings){
            this.settings[setting].enabled = this.settings[setting].default;
        }
        this.items.Sword.equipped = true;
        this.items.Pants.equipped = true;
        this.presets.p1 = [];
        this.presets.p2 = [];
        this.presets.p3 = [];
        this.resetCombat();
    },
    getItemOrder: function(){
        var items = [];
        for (var item in this.items){
            items.push({name: item, zone: (this.items[item].zone) ? this.items[item].zone : 0})
        }
        function itemSort(a,b){
            if (a.zone > b.zone) return 1;
	        if (a.zone < b.zone) return -1;
        }
        items.sort(itemSort);
        return items;
    },
    getContracts: function(){
        var items = this.getItemOrder();
        var contracts = [];
        for (var x = 0; x < items.length; x++){
            if (!this.items[items[x].name].owned) {
                contracts.push(items[x].name)
                if (contracts.length >= 3) return contracts;
            }
        }
        return contracts;
    },
    contractPrice: function(item){
        var itemObj = this.items[item];
        var dif = itemObj.zone - 75
        var total = (100 * Math.pow(1.2023, dif));
        if (itemObj.dustType == "shards") total /= 1e9;
        return total;
    },
    oneTimerPrice: function(item){
        var itemObj = this.oneTimers[item];
        var allItems = this.getItemOrder();
        var index = itemObj.requiredItems - 1;
        if (itemObj.useShards) index++;
        if (index <= 6) return 10000;
        var lastItem = allItems[index];
        var contractPrice = this.contractPrice(lastItem.name);
        if (itemObj.useShards) return Math.ceil(contractPrice / 2);
        return Math.ceil(contractPrice * 1000) / 10;
    },
    toggleSetting: function(which){
        var setting = this.settings[which];
        if (setting.enabled == setting.text.length - 1) setting.enabled = 0;
        else setting.enabled++;
        if (setting.onToggle) setting.onToggle();
        this.popup(true, false, true);
    },
    settings: {
        loadHide: {
            enabled: 1,
            default: 1,
            text: ["Leave Items on Preset Load", "Hide Unused Items on Preset Load"]
        },
        loadLevel: {
            enabled: 1,
            default: 1,
            text: ["Leave Enemy Level on Preset Load", "Set Enemy Level on Preset Load"]
        },
        loadRing: {
            enabled: 1,
            default: 1,
            text: ["Leave Ring on Preset Load", "Load Ring Mods on Preset Load"],
            hideUnless: function(){
                return (autoBattle.oneTimers.The_Ring.owned);
            }
        },
        practice: {
            enabled: 0,
            default: 0,
            text: ["Practice Off - 10x Speed, no Dust or Progress", "Practice On - 10x Speed, no Dust or Progress"],
            onToggle: function(){
                if (this.enabled) autoBattle.speed = 10;
                else autoBattle.speed = 1;
                autoBattle.resetCombat(true);
            }
        }
    },
    items: {
        //Starting items
        Menacing_Mask: {
            owned: true,
            equipped: false,
            hidden: false,
            level: 1,
            description: function(){
                return "-" + prettify((1 - this.trimpAttackSpeed()) * 100) + "% Huffy Attack Time, +" + prettify((1 - this.enemyAttackSpeed()) * -100) + "% Enemy Attack Time.";
            },
            upgrade: "-2% Huffy Attack Time, +2% Enemy Attack Time (compounding)",
            trimpAttackSpeed: function(){
                return Math.pow(0.98, this.level);
            },
            enemyAttackSpeed: function(){ 
                return 1.05 * Math.pow(1.02, (this.level - 1));
            },
            doStuff: function(){
                autoBattle.trimp.attackSpeed *= this.trimpAttackSpeed();
                autoBattle.enemy.attackSpeed *= this.enemyAttackSpeed();
            },
            priceMod: 5
        },
        Sword: {
            owned: true,
            equipped: true,
            hidden: false,
            level: 1,
            description: function(){
                return "+" + this.effect() + " attack damage.";
            },
            upgrade: "+1 attack damage",
            effect: function(){
                return this.level;
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.effect();
            },
            priceMod: 2.5
        },
        Armor: {
            owned: true,
            equipped: false,
            hidden: false,
            level: 1,
            description: function(){
                return "+" + prettify(this.effect()) + " base health.";
            },
            upgrade: "+20 base health",
            effect: function(){
                return 20 * this.level;
            },
            doStuff: function(){
                autoBattle.trimp.maxHealth += this.effect();
            },
            priceMod: 5
        },
        Rusty_Dagger: {
            owned: true,
            equipped: false,
            hidden: false,
            level: 1,
            description: function(){
                return "Can create a Bleed on the Enemy for 10 seconds. +" + prettify(this.bleedMod() * 100) + "% Bleed Damage" + ((this.level >= 5) ? ", +" + prettify(this.attack()) + " Attack" : "") + ". +" + prettify(this.bleedChance()) + "% Bleed Chance, doubled if the Enemy is Shocked or Poisoned.";
            },
            upgrade: "+10 Attack and +20% Bleed Damage per 5 levels. +5% Bleed Damage and +3% Bleed Chance",
            attack: function(){
                return Math.floor(this.level / 5) * 10;
            },
            bleedChance: function(){
                return 17 + (3 * this.level);
            },
            bleedMod: function(){
                var val = 0.1 + (0.05 * this.level);
                val += (Math.floor(this.level / 5) * 0.2);
                return val;
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.bleedMod += this.bleedMod();
                if (autoBattle.trimp.bleedTime < 10000) autoBattle.trimp.bleedTime = 10000;
                autoBattle.trimp.bleedChance += (autoBattle.enemy.poison.time > 0 || autoBattle.enemy.shock.time > 0) ? (this.bleedChance() * 2) : this.bleedChance();
            },
            startPrice: 25,
            priceMod: 4
        },
        Fists_of_Goo: {
            owned: true,
            equipped: false,
            hidden: false,
            level: 1,
            description: function(){
                return "Can create a Poison on the Enemy for 10 seconds. +" + prettify(this.effect()) + " Poison Damage. +25% Poison Chance, doubled if the enemy is bleeding or shocked."
            },
            upgrade: "+1 poison damage",
            effect: function(){
                return this.level;
            },
            doStuff: function(){
                autoBattle.trimp.poisonMod += this.effect();
                autoBattle.trimp.poisonChance += (autoBattle.enemy.shock.time > 0 || autoBattle.enemy.bleed.time > 0) ? 50 : 25;
                if (autoBattle.trimp.poisonTime < 10000) autoBattle.trimp.poisonTime = 10000;
            },
            priceMod: 6,
            startPrice: 50
        },
        Battery_Stick: {
            owned: true,
            equipped: false,
            hidden: false,
            level: 1,
            description: function(){
                return "Can create a Shock on the Enemy for 10 seconds. +" + prettify(this.shockMod() * 100) + "% Shock Damage. +35% Shock Chance, doubled if the enemy is bleeding or poisoned.";
            },
            upgrade: "+10% Shock Damage",
            shockMod: function(){
                return 0.15 + (0.1 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.shockChance += (autoBattle.enemy.bleed.time > 0 || autoBattle.enemy.poison.time > 0) ? 70 : 35;
                autoBattle.trimp.shockMod += this.shockMod();
                autoBattle.trimp.shockTime = 10000;
            },
            startPrice: 25,
            priceMod: 4
        },
        Pants: {
            owned: true,
            equipped: true,
            hidden: false,
            level: 1,
            description: function(){
                return "+" + prettify(this.effect()) + " Defense."
            },
            upgrade: "+1 Defense",
            effect: function(){
                return this.level;
            },
            doStuff: function(){
                autoBattle.trimp.defense += this.effect();
            },
        },
        //unlockables

        //raincoat, 75
        //pouch 78
        Chemistry_Set: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 81,
            description: function(){
                var stacks = this.poisonStack();
                return "+50% Poison Chance if the Enemy is not already Poisoned. +" + this.defenseEffect() + " Defense if the Enemy is Poisoned. +" + prettify(this.poisonChance()) + "% Poison Chance. Poisons you inflict can stack " + stacks + " more time" + needAnS(stacks) + ".";
            },
            upgrade: "+1 Max Poison Stack per 4 levels. +1 Defense, +4% standard Poison Chance",
            defenseEffect: function(){
                return this.level;
            },
            poisonChance: function(){
                return 6 + (this.level * 4);
            },
            poisonStack: function(){
                var levels = Math.floor(this.level / 4);
                return 1 + levels;
            },
            doStuff: function(){
                if (autoBattle.enemy.poison.time > 0) autoBattle.trimp.defense += this.defenseEffect();
                else autoBattle.trimp.poisonChance += 50;
                autoBattle.trimp.poisonChance += this.poisonChance();
                autoBattle.trimp.poisonStack += this.poisonStack();
            },
            priceMod: 4,
            startPrice: 200
        },
        //bad medkit - 84
        Comfy_Boots: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 87,
            description: function(){
                return "+" + prettify(this.defense()) + " Defense. +" + prettify(this.resistance()) + "% to all Resistances.";
            },
            upgrade: "+2 Defense, +5% Resist",
            defense: function(){
                return 2 + (this.level * 2);
            },
            resistance: function(){
                return (this.level * 5);
            },
            doStuff: function(){
                autoBattle.trimp.defense += this.defense();
                var res = this.resistance();
                autoBattle.trimp.bleedResist += res;
                autoBattle.trimp.poisonResist += res;
                autoBattle.trimp.shockResist += res;
            },
            startPrice: 430
        },
        //Labcoat 90
        Lifegiving_Gem: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 93,
            description: function(){
                return "Increases Dust gained from Enemies by " + prettify(this.effect() * 100) + "% PLUS your Lifesteal amount when the Enemy dies."
            },
            upgrade: "+10% Dust Gained",
            effect: function(){
                return 0.2 + (0.1 * this.level);
            },
            dustIncrease: function(){
                return this.effect() + Math.max(0, (autoBattle.trimp.lifesteal - autoBattle.enemy.lifestealResist));
            },
            startPrice: 650,
            priceMod: 4
        },
        Mood_Bracelet: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 95,
            description: function(){
                return "-" + prettify((1 - this.effect()) * 100) + "% Attack Time and +" + prettify(this.defense()) + " Defense if the Enemy is not Bleeding."
            },
            upgrade: "-3% Attack Time (compounding), +4 Defense",
            effect: function(){
                return 0.8765 * Math.pow(0.97, this.level);
            },
            defense: function(){
                return 6 + (4 * this.level);
            },
            doStuff: function(){
                if (autoBattle.enemy.bleed.time <= 0){
                    autoBattle.trimp.attackSpeed *= this.effect();
                    autoBattle.trimp.defense += this.defense();
                }
            },
            priceMod: 4,
            startPrice: 1100
        },
        Hungering_Mold: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 98,
            description: function(){
                return "Heal for " + prettify(this.healAmt()) + " per stack of Poison whenever one of your Poisons deals damage. Your Poisons tick " + prettify((1 - this.tickMult()) * 100) + "% faster.";
            },
            upgrade: "+0.5 Heal on Poison Tick, +1% Poison Tick Speed",
            healAmt: function(){
                return 0.5 + (0.5 * this.level);
            },
            tickMult: function(){
                return 0.909 * Math.pow(0.99, this.level);
            },
            doStuff: function(){
                autoBattle.trimp.poisonTick *= this.tickMult();
                autoBattle.trimp.poisonHeal += this.healAmt();
            },
            priceMod: 5,
            startPrice: 2000
        },
        Recycler: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 100,
            description: function(){
                return "+" + prettify(this.effect() * 100) + "% Lifesteal. Huffy's Lifesteal heals twice as much off of Bleed damage.";
            },
            upgrade: "+5% Lifesteal",
            effect: function(){
                return 0.2 + (0.05 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.lifesteal += this.effect();
            },
            startPrice: 2800,
            priceMod: 5
        },
        Shining_Armor: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 103,
            description: function(){
                return "+" + prettify(this.defense()) + " Defense. +" + prettify(this.health()) + " Health.";
            },
            upgrade: "+6 defense, +100 health",
            defense: function(){
                return 14 + (6 * this.level);
            },
            health: function(){
                return 200 + (this.level * 100);
            },
            doStuff: function(){
                autoBattle.trimp.defense += this.defense();
                autoBattle.trimp.maxHealth += this.health();
            },
            priceMod: 5,
            startPrice: 4000
        },
        Shock_and_Awl:{
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 105,
            description: function(){
                return "Can create a Shock on an enemy for 20 seconds. +" + prettify(this.attack()) + " Attack, +" + prettify(this.shockChance()) + "% Shock Chance, +" + prettify(this.shockMod() * 100) + "% Shock Damage. -25% Attack Time if the Enemy is not Shocked, +25% Lifesteal if the Enemy is Shocked.";
            },
            upgrade: "+4 Attack, +10% Shock Chance, +10% Shock Damage",
            attack: function(){
                return 6 + (4 * this.level);
            },
            shockChance: function(){
                return 20 + (10 * this.level);
            },
            shockMod: function(){
                return .40 + (.1 * this.level);
            },
            doStuff: function(){
                if (autoBattle.trimp.shockTime < 20000) autoBattle.trimp.shockTime = 20000;
                autoBattle.trimp.shockMod += this.shockMod();
                autoBattle.trimp.shockChance += this.shockChance();
                autoBattle.trimp.attack += this.attack();
                if (autoBattle.enemy.shock.time <= 0) autoBattle.trimp.attackSpeed *= 0.75;
                else autoBattle.trimp.lifesteal += 0.25;
            },
            priceMod: 5,
            startPrice: 5750
        },
        //spiked gloves - 108
        Tame_Snimp: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 110,
            description: function(){
                return "Can create a Poison on the Enemy for 10 seconds. +" + prettify(this.poisonChance()) + "% Poison Chance, +" + prettify(this.poisonMod()) + " Poison Damage. Enemy Attack is reduced by 15% while the Enemy is Poisoned.";
            },
            upgrade: "+10% Poison Chance, +2 Poison Damage",
            poisonChance: function(){
                return 30 + (10 * this.level);
            },
            poisonMod: function(){
                return 5 + (2 * this.level);
            },
            doStuff: function(){
                if (autoBattle.enemy.poison.time > 0) autoBattle.enemy.attack *= 0.85;
                if (autoBattle.trimp.poisonTime < 10000) autoBattle.trimp.poisonTime = 10000;
                autoBattle.trimp.poisonChance += this.poisonChance();
                autoBattle.trimp.poisonMod += this.poisonMod();
            },
            priceMod: 5.5,
            startPrice: 15000
        },
        Lich_Wraps: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 113,
            description: function(){
                return "When Poisoned, Bleeding, or Shocked, gain +" + prettify(this.attack()) + " Attack, -15% Attack Time, +" + prettify(this.lifesteal() * 100) + "% Lifesteal, and take " + prettify((1 - this.damageTakenMult()) * 100) + "% less damage from all sources.";
            },
            upgrade: "+6 Attack, +6% Lifesteal, -3% damage taken (compounding, never reaching 75%)",
            damageTakenMult: function(){
                return (((0.825 * Math.pow(0.93, (this.level - 1))) / 1.5) + 0.25);
            },
            attack: function(){
                return 9 + (6 * this.level);
            },
            lifesteal: function(){
                return 0.09 + (0.06 * this.level);
            },
            doStuff: function(){
                if (autoBattle.trimp.bleed.time > 0 || autoBattle.trimp.shock.time > 0 || autoBattle.trimp.poison.time > 0){
                    autoBattle.trimp.damageTakenMult *= this.damageTakenMult();
                    autoBattle.trimp.attack += this.attack();
                    autoBattle.trimp.lifesteal += this.lifesteal();
                    autoBattle.trimp.attackSpeed *= 0.85;
                }
            },
            priceMod: 4,
            startPrice: 25000

        },
        Wired_Wristguards: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 115,
            description: function(){
                return "+" + prettify(this.defense()) + " Defense, +" + prettify(this.shockChance()) + "% Shock Chance, +" + prettify(this.shockMod() * 100) + "% Shock Damage, +50% to all Resistances. If the Enemy is Shocked, increase its Attack Time by " + prettify((this.enemySpeed() - 1) * 100) + "%.";
            },
            upgrade: "+3 Defense, +15% Shock Chance, +15% Shock Damage, +2% Enemy Attack Time",
            defense: function(){
                return 7 + (3 * this.level);
            },
            shockChance: function(){
                return 25 + (15 * this.level);
            },
            shockMod: function(){
                return 0.25 + (0.15 * this.level);
            },
            enemySpeed: function(){
                return 1.18 + (0.02 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.defense += this.defense();
                autoBattle.trimp.shockChance += this.shockChance();
                autoBattle.trimp.shockMod += this.shockMod();
                autoBattle.trimp.shockResist += 50;
                autoBattle.trimp.poisonResist += 50;
                autoBattle.trimp.bleedResist += 50;
                if (autoBattle.enemy.shock.time > 0){
                    autoBattle.enemy.attackSpeed *= this.enemySpeed();
                }
            },
            startPrice: 44000,
            priceMod: 4.5
        },

        Sword_and_Board: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 120,
            description: function(){
                return "+" + prettify(this.attack()) + " Attack, +" + prettify(this.defense()) + " Defense, +" + prettify(this.health()) + " Health, +" + prettify(this.resists()) + "% to all Resistances.";
            },
            upgrade: "+5 Attack, +50 Health, +4 Defense, +10% Resists",
            attack: function(){
                return 10 + (5 * this.level);
            },
            defense: function(){
                return 6 + (4 * this.level);
            },
            health: function(){
                return 350 + (50 * this.level);
            },
            resists: function(){
                return 10 + (10 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.defense += this.defense();
                autoBattle.trimp.maxHealth += this.health();
                var resists = this.resists();
                autoBattle.trimp.shockResist += resists;
                autoBattle.trimp.poisonResist += resists;
                autoBattle.trimp.bleedResist += resists;
            },
            priceMod: 5,
            startPrice: 90000
        },
        Bilious_Boots: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 122,
            description: function(){
                return "+" + prettify(this.poisonMod()) + " Poison Damage, +1 Max Poison Stack, +" + prettify(this.health()) + " Health, +" + prettify(this.resists()) + "% to all Resistances.";
            },
            upgrade: "+3 Poison Damage, +50 Health, +10% Resists",
            poisonMod: function(){
                return 7 + (3 * this.level);
            },
            health: function(){
                return 150 + (50 * this.level);
            },
            resists: function(){
                return 10 + (10 * this.level);
            },
            doStuff: function(){
                var resists = this.resists();
                autoBattle.trimp.shockResist += resists;
                autoBattle.trimp.poisonResist += resists;
                autoBattle.trimp.bleedResist += resists;
                autoBattle.trimp.poisonMod += this.poisonMod();
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.poisonStack++;
            },
            priceMod: 5,
            startPrice: 100000
        },
        Bloodstained_Gloves: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 123,
            description: function(){
                return "+" + prettify(this.bleedChance()) + "% to Bleed Chance, +" + prettify(this.attack()) + " Attack, -25% Enemy Attack Time, -25% Enemy Attack Damage. Fills up " + prettify(this.barFill() * 100) + "% of your Attack Speed bar whenever you cause or receive a Bleed.";
            },
            upgrade: "+5% Bleed Chance, +2 Attack, +5% (up to +160%) bar filled on Bleed",
            attack: function(){
                return 6 + (this.level * 2)
            },
            onBleed: function(){
                autoBattle.trimp.lastAttack += (autoBattle.trimp.attackSpeed * this.barFill());
            },
            bleedChance: function(){
                return 25 + (5 * this.level);
            },
            barFill: function(){
                var fill = 0.20 + (0.05 * this.level);
                if (fill > 1.6) fill = 1.6;
                return fill;
            },
            doStuff: function(){
                autoBattle.trimp.bleedChance += this.bleedChance();
                autoBattle.enemy.attackSpeed *= 0.75;
                autoBattle.enemy.attack *= 0.75;
                autoBattle.trimp.attack += this.attack();
            },
            startPrice: 160000
        },
        Unlucky_Coin: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 125,
            description: function(){
                return "+" + prettify(this.attack()) + " Attack. +" + prettify(this.lifesteal() * 100) + "% Lifesteal if the Enemy is not Poisoned or Bleeding.";
            },
            upgrade: "+4 Attack, +10% Lifesteal",
            attack: function(){
                return 11 + (this.level * 4);
            },
            lifesteal: function(){
                return 0.2 + (this.level * .1);
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.attack();
                if (autoBattle.enemy.bleed.time <= 0 && autoBattle.enemy.poison.time <= 0){
                    autoBattle.trimp.lifesteal += this.lifesteal();
                }
            },
            priceMod: 5,
            startPrice: 400000
        },
        Eelimp_in_a_Bottle: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 130,
            description: function(){
                return "+" + prettify(this.shockChance()) + "% Shock Chance, +" + prettify(this.shockMod() * 100) + "% Shock Damage, +" + prettify(this.shockResist()) + "% Shock Resist. -" + prettify((1 - this.attackSpeed()) * 100) + "% Attack Time if the Enemy is Shocked. When you Shock an Enemy, they lose all progress towards their attack. +" + prettify(this.attack()) + " Attack for each time you've Shocked this Enemy (up to 10 times).";
            },
            upgrade: "+5% Shock Chance, +5% Shock Damage, -5% Attack Time, +5% Shock Resist, +1 Attack per Shock",
            attackSpeed: function(){
                return 0.9 * Math.pow(0.95, this.level);
            },
            shockChance: function(){
                return 35 + (5 * this.level);
            },
            shockMod: function(){
                return .65 + (.1 * this.level);
            },
            shockResist: function(){
                return 10 + (5 * this.level);
            },
            attack: function(){
                return 2 + this.level;
            },
            doStuff: function(){
                autoBattle.trimp.shockMod += this.shockMod();
                autoBattle.trimp.shockChance += this.shockChance();
                if (autoBattle.enemy.shock.time >= 0) autoBattle.trimp.attackSpeed *= this.attackSpeed();
                autoBattle.trimp.attack += (Math.min(10, autoBattle.enemy.shock.count) * this.attack());
                autoBattle.trimp.shockResist += this.shockResist();
            },
            priceMod: 5,
            startPrice: 1000000
        },        
        Big_Cleaver: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 133,
            description: function(){
                return "Can create a Bleed on the Enemy for 10 seconds. +100% Bleed Chance if the Enemy is at full Health, otherwise +25%. +" + prettify(this.attack()) + " Attack if the Enemy is Bleeding. +" + prettify(this.bleedMod() * 100) + "% Bleed Damage, +" + prettify(this.health()) + " Health.";
            },
            upgrade: "+2 Attack, +25% Bleed Damage, +100 Health",
            attack: function(){
                return 30 + (this.level * 2);
            },
            bleedMod: function(){
                return 1 + (0.25 * this.level);
            },
            health: function(){
                return 500 + (100 * this.level);
            },
            doStuff: function(){
                if (autoBattle.enemy.health == autoBattle.enemy.maxHealth) autoBattle.trimp.bleedChance += 100;
                else autoBattle.trimp.bleedChance += 25;
                autoBattle.trimp.maxHealth += this.health();
                if (autoBattle.enemy.bleed.time > 0) autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.bleedMod += this.bleedMod();
                if (autoBattle.trimp.bleedTime <= 10000) autoBattle.trimp.bleedTime = 10000;
            },
            priceMod: 4,
            startPrice: 3000000
        },
        The_Globulator: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 135,
            description: function(){
                return "+" + prettify(this.defense()) + " Defense and +" + prettify(this.health()) + " Max Health if the Enemy is Poisoned. On adding a new Poison Stack to an Enemy that hasn't had poisons expire, heal for half of this item's Max Health. If the Enemy is at Max Poison Stacks, non-Lifesteal healing effects on you are doubled. +" + prettify(this.poisonMod()) + " Poison Damage."
            },
            upgrade: "+5 Defense, +500 Health, +10 Poison Damage",
            defense: function(){
                return 25 + (5 * this.level);
            },
            health: function(){
                return 500 + (500 * this.level);
            },
            poisonMod: function(){
                return 15 + (10 * this.level);
            },
            onPoisonStack: function(stacks){
                if (stacks == 1) autoBattle.trimp.maxHealth += this.health();
                if (autoBattle.enemy.poison.expired) return;
                autoBattle.trimp.health += (this.health() / 2);
                if (autoBattle.trimp.health > autoBattle.trimp.maxHealth) autoBattle.trimp.health = autoBattle.trimp.maxHealth;
            },
            doStuff: function(){
                if (autoBattle.enemy.poison.time > 0){
                    autoBattle.trimp.maxHealth += this.health();
                    autoBattle.trimp.defense += this.defense();
                }
                autoBattle.trimp.poisonMod += this.poisonMod();
            },
            startPrice: 5e6,
            priceMod: 10
        },
        Metal_Suit: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 138,
            description: function(){
                return "+" + prettify(this.defense()) + " Defense, +" + prettify(this.health()) + " Health, +" + prettify(this.resist()) + "% Bleed Resist. If Huffy has an item that can create a Bleed, gain +" + prettify(this.attack()) + " Attack."
            },
            upgrade: "+30 Defense, +1000 Health, +20% Bleed Resist, +5 Attack",
            attack: function(){
                return 10 + (5 * this.level);
            },
            defense: function(){
                return (-10 + (this.level * 30));
            },
            health: function(){
                return (-500 + (this.level * 1000));
            },
            resist: function(){
                return 30 + (20 * this.level);
            },
            doStuff: function(){
                if (autoBattle.items.Rusty_Dagger.equipped || autoBattle.items.Big_Cleaver.equipped) autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.defense += this.defense();
                autoBattle.trimp.bleedResist += this.resist();
            },
            priceMod: 10,
            startPrice: 6e6
        },
        Nozzled_Goggles: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 140,
            description: function(){
                return "The Enemy is always Shocked, taking at least " + prettify(this.shockMod() * 100) + "% more damage. +" + prettify(this.health()) + " Health, +" + prettify(this.resist()) +  "% Poison Resist, +3 Maximum Poison Stacks.";
            },
            upgrade: "+20% PermaShock Damage, +1000 Health, 20% Poison Resist",
            shockMod: function(){
                return (0.2 * this.level);
            },
            health: function(){
                return (-500 + (this.level * 1000));
            },
            resist: function(){
                return (20 * this.level);
            },
            doStuff: function(){
                var enemy = autoBattle.enemy;
                if (enemy.shock.time <= 0 || enemy.shock.mod < this.shockMod()){
                    enemy.shock.time = 9999999;
                    enemy.shock.mod = this.shockMod();
                }
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.poisonResist += this.resist();
                autoBattle.trimp.poisonStack += 3;
            },
            startPrice: 7e6,
            priceMod: 10
        },
        Sundering_Scythe: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 143,
            description: function(){
                return "-" + prettify((1 - this.attackTime()) * 100) + "% Attack Time, +" + prettify(this.attack()) + " Attack, +" + prettify(this.health()) + " Health, +" + prettify(this.lifesteal() * 100) + "% Lifesteal. Your Shocks last a maximum of 10 seconds, and your Bleeds can be reapplied when below 5 seconds.";
            },
            upgrade: "-5% Attack Time, +5 Attack, +250 Health, +5% Lifesteal",
            attackTime: function(){
                return 0.842 * Math.pow(0.95, this.level);
            },
            attack: function(){
                return 15 + (5 * this.level);
            },
            health: function(){
                return 500 + (250 * this.level);
            },
            lifesteal: function(){
                return 0.15 + (0.05 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.attackSpeed *= this.attackTime();
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.lifesteal += this.lifesteal();
            },
            startPrice: 15e6,
            priceMod: 10
        },
        //Shank 145
        Plague_Bringer: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 148,
            description: function(){
                return "Your Poisons tick +" + prettify((1 - this.tickMult()) * 100) + "% faster. +" + prettify(this.eachPoison()) + " Poison Damage for every percentage point of Bleed or Shock resist on the Enemy. Heal for " + prettify(this.healAmt()) + " per stack of Poison when your Poisons deal damage." 
            },
            upgrade: "+2% Poison Tick Rate, +0.05 Poison Damage per Enemy Resist, +5 Heal on Poison Tick",
            tickMult: function(){
                return 0.816 * Math.pow(0.98, this.level);
            },
            eachPoison: function(){
                return 0.05 + (0.05 * this.level);
            },
            healAmt: function(){
                return 5 + (5 * this.level);
            },
            poisonMod: function(){
                var res = autoBattle.enemy.bleedResist + autoBattle.enemy.shockResist;
                return Math.floor(res * this.eachPoison());
            },
            doStuff: function(){
                autoBattle.trimp.poisonMod += this.poisonMod();
                autoBattle.trimp.poisonTick *= this.tickMult();
                autoBattle.trimp.poisonHeal += this.healAmt();
            },
            startPrice: 70e6,
            priceMod: 10
        },
        Very_Large_Slime: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 150,
            description: function(){
                return "Can create a Poison for 20 seconds. +" + prettify(this.poisonChance()) + "% Poison Chance, +" + prettify(this.poisonMod()) + " Poison Damage, +" + prettify(this.defense()) + " Defense, +" + prettify(this.health()) + " Health. Every third hit against an Enemy with Max Poison Stacks adds another Max Stack (up to +" + prettify(this.maxStacks()) + " Max Stacks)."; 
            },
            upgrade: "+15% Poison Chance, +20 Poison Damage, +50 Defense, +500 Health, +2 Max Stacks",
            poisonChance: function(){
                return 35 + (15 * this.level);
            },
            poisonMod: function(){
                return 10 + (20 * this.level);
            },
            defense: function(){
                return 50 + (50 * this.level);
            },
            health: function(){
                return 500 + (500 * this.level);
            },
            maxStacks: function(){
                return 8 + (this.level * 2);
            },
            doStuff: function(){
                autoBattle.trimp.poisonChance += this.poisonChance();
                autoBattle.trimp.poisonMod += this.poisonMod();
                if (autoBattle.trimp.poisonTime < 20000) autoBattle.trimp.poisonTime = 20000;
                autoBattle.trimp.defense += this.defense();
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.poisonStack += Math.min(this.maxStacks(), Math.floor(autoBattle.enemy.poison.hitsAtMax / 3));
            },
            startPrice: 100e6,
            priceMod: 10
        },
        //Monkimp Paw, 155
        Grounded_Crown: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 160,
            description: function(){
                return "+" + prettify(this.attack()) + " Attack, +" + prettify(this.shockMod() * 100) + "% Shock Damage, +" + prettify(this.defense()) + " Defense, +" + prettify(this.health()) + " Health. If the Enemy is Poisoned or Bleeding, Huffy loses 20% of his Max Health per second.";
            },
            upgrade: "+50 Attack, +50% Shock Damage, +50 Defense, +1000 Health",
            attack: function(){
                return 50 + (50 * this.level);
            },
            shockMod: function(){
                return 0.5 + (0.5 * this.level);
            },
            defense: function(){
                return (50 * this.level);
            },
            health: function(){
                return 500 + (1000 * this.level);
            },
            afterCheck: function(){
                if (autoBattle.enemy.poison.time > 0 || autoBattle.enemy.bleed.time > 0){
                    var mod = 20 / autoBattle.frameTime;
                    autoBattle.trimp.health -= (autoBattle.trimp.maxHealth * mod * autoBattle.trimp.damageTakenMult);
                    if (autoBattle.trimp.health < 0.01) autoBattle.trimp.health = 0;
                }
            },
            doStuff: function(){
                
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.shockMod += this.shockMod();
                autoBattle.trimp.defense += this.defense();
                autoBattle.trimp.maxHealth += this.health();
            },
            startPrice: 650e6,
            priceMod: 10
        },
        Fearsome_Piercer: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 165,
            description: function(){
                return "+" + prettify(this.attack()) + " Attack, +" + prettify(this.lifesteal() * 100) + "% Lifesteal, +" + prettify(this.bleedMod() * 100) + "% Bleed Damage, +" + prettify(this.bleedChance()) + "% Bleed Chance. If you have an item that can cause a Bleed, the Enemy starts combat with 25% less Max Health."
            },
            upgrade: "+75 Attack, +10% Lifesteal, +75% Bleed Damage, +50% Bleed Chance",
            attack: function(){
                return 125 + (75 * this.level);
            },
            lifesteal: function(){
                return 0.3 + (0.1 * this.level);
            },
            bleedMod: function(){
                return 0.25 + (0.75 * this.level);
            },
            bleedChance: function(){
                return 25 + (50 * this.level);
            },
            onEnemy: function(){
                if (autoBattle.items.Rusty_Dagger.equipped || autoBattle.items.Big_Cleaver.equipped || autoBattle.items.Bag_of_Nails.equipped || autoBattle.items.Doppelganger_Diadem.equipped){
                    autoBattle.enemy.baseHealth *= 0.75;
                    autoBattle.enemy.maxHealth *= 0.75;
                    autoBattle.enemy.health = autoBattle.enemy.maxHealth;
                }
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.lifesteal += this.lifesteal();
                autoBattle.trimp.bleedMod += this.bleedMod();
                autoBattle.trimp.bleedChance += this.bleedChance();
            },
            startPrice: 1.5e9,
            priceMod: 10
        },
        Bag_of_Nails: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 168,
            startPrice: 2.7e9,
            priceMod: 10,
            description: function(){
                return "Can create a Bleed on the Enemy for 10 seconds. Enemies are unaffected by your Slow Aura, but deal 25% less damage while Bleeding. +" + prettify(this.attack()) + " Attack, +" + prettify(this.bleedMod() * 100) + "% Bleed Damage, +" + prettify(this.health()) + " Health."
            },
            upgrade: "+100 Attack, +75% Bleed Damage, +500 Health",
            attack: function(){
                return 150 + (this.level * 100);
            },
            bleedMod: function(){
                return 1.25 + (0.75 * this.level);
            },
            health: function(){
                return 500 + (500 * this.level);
            },
            doStuff: function(){
                if (autoBattle.enemy.bleed.time > 0){
                    autoBattle.enemy.attack *= 0.75;
                    autoBattle.enemy.noSlow = true;
                }
                else autoBattle.enemy.noSlow = false;
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.bleedMod += this.bleedMod();
                autoBattle.trimp.maxHealth += this.health();
                if (autoBattle.trimp.bleedTime <= 10000) autoBattle.trimp.bleedTime = 10000;
            }
        },
        Blessed_Protector: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 170,
            description: function(){
                return "+" + prettify(this.health()) + " Health, +" + prettify(this.defense()) + " Defense, +" + prettify(this.lifesteal() * 100) + "% Lifesteal. Huffy gains 0.5% increased Attack for each % of his missing Health. When Huffy is below 50% Health, take 30% less damage from all sources. " 
            },
            upgrade: "+1000 Health, +100 Defense, +25% Lifesteal",
            health: function(){
                return 1000 + (1000 * this.level);
            },
            defense: function(){
                return 100 + (100 * this.level);
            },
            lifesteal: function(){
                return 0.25 + (0.25 * this.level);
            },
            afterCheck: function(){
                if (autoBattle.trimp.health <= 0) return;
                var healthPct = autoBattle.trimp.health / autoBattle.trimp.maxHealth;
                if (healthPct < 0.5){
                    autoBattle.trimp.damageTakenMult *= 0.7;
                }
                if (healthPct < 1){
                    var boost = (1 - healthPct);
                    boost = (1 + (boost * 0.5));
                    autoBattle.trimp.attack *= boost;
                }
            },
            doStuff: function(){
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.defense += this.defense();
                autoBattle.trimp.lifesteal += this.lifesteal();
            },
            startPrice: 4e9,
            priceMod: 10
        },
        The_Doomspring:{
            description: function(){
                var stack = prettify((1 - this.attackTime()) * 100);
                return "+" + prettify(this.health()) + " Health, -" + stack + "% Attack Time. For every 15000 damage taken this battle, -" + stack + "% more Attack Time. Stacks up to " + this.stacks() + " times.";
            },
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 180,
            upgrade: "-5% Attack Time, +1500 Health, +1 stacks",
            attackTime: function(){
                return 0.842 * Math.pow(0.95, this.level);
            },
            health: function(){
                return 1500 + (1500 * this.level);
            },
            stacks: function(){
                return 2 + this.level;
            },
            doStuff: function(){
                var stacks = Math.floor(autoBattle.trimp.dmgTaken / 15000) + 1;
                var maxStacks = this.stacks();
                if (stacks > maxStacks) stacks = maxStacks;
                autoBattle.trimp.attackSpeed *= Math.pow(this.attackTime(), stacks);
                autoBattle.trimp.maxHealth += this.health();
            },
            dustType: "shards",
            startPrice: 22,
            priceMod: 15
        },
        Snimp__Fanged_Blade: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 185,
            description: function(){
                return "+" + prettify(this.attack()) + " Attack, +" + this.poisonStack() + " Max Poison Stacks. If the enemy is Poisoned, +" + prettify(this.bleedMod() * 100) + "% Bleed Damage. If the enemy is Bleeding, +" + prettify(this.poisonMod()) + " Poison Damage.";
            },
            upgrade: "+250 Attack, +5 Max Poison Stacks, +200% Bleed Damage, +250 Poison Damage",
            attack: function(){
                return 250 + (250 * this.level);
            },
            poisonMod: function(){
                return 250 * this.level;
            },
            bleedMod: function(){
                return 1 + (2 * this.level);
            },
            poisonStack: function(){
                return 5 * this.level;
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.poisonStack += this.poisonStack();
                if (autoBattle.enemy.poison.time > 0) autoBattle.trimp.bleedMod += this.bleedMod();
                if (autoBattle.enemy.bleed.time > 0) autoBattle.trimp.poisonMod += this.poisonMod();
            },
            dustType: "shards",
            startPrice: 159,
            priceMod: 15
        },
        //Dopp signet 190
        Wrath_Crafted_Hatchet:{
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 195,
            description: function(){
                return "+" + prettify(this.attack()) + " Attack, +" + prettify(this.health()) + " Health, +" + prettify(this.defense()) + " Defense, -" + prettify((1 - this.attackTime()) * 100) + "% Attack Time.";
            },
            upgrade: "+750 Attack, +2000 Health, +200 Defense, -5% Attack Time",
            attack: function(){
                return 1250 + (750 * this.level);
            },
            attackTime: function(){
                return 0.842 * Math.pow(0.95, this.level);
            },
            health: function(){
                return 2000 + (2000 * this.level);
            },
            defense: function(){
                return 200 + (200 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.attackSpeed *= this.attackTime();
                autoBattle.trimp.defense += this.defense();
            },
            dustType: "shards",
            startPrice: 400,
            priceMod: 15
        },
        //basket of souls 200
        Goo_Golem: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 205,
            description: function(){
                return "+" + prettify(this.health()) + " Health, +" + prettify(this.poisonMod()) + " Poison Damage, +" + prettify(this.poisonStack()) + " Max Poison Stacks. If the Enemy is Poisoned, all damage Huffy takes is stored in the Golem, and Huffy takes 10% of the Golem's stored damage every second.";
            },
            upgrade: "+1000 Health, +400 Poison Damage, +5 Max Poison Stacks",
            health: function(){
                return 2000 + (1000 * this.level);
            },
            poisonMod: function(){
                return 400 * this.level;
            },
            poisonStack: function(){
                return 5 + (5 * this.level);
            },
            active: function(){
                if (this.equipped && autoBattle.enemy.poison.time > 0) return true;
                return false;
            },
            doStuff: function(){
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.poisonMod += this.poisonMod();
                autoBattle.trimp.poisonStack += this.poisonStack();
                if (autoBattle.battleTime > autoBattle.trimp.lastGoo + 1000){     
                    if (autoBattle.trimp.lastGoo == -1) autoBattle.trimp.lastGoo = autoBattle.battleTime;
                    else autoBattle.trimp.lastGoo += 1000;
                    var dmg = autoBattle.trimp.gooStored * 0.1;
                    autoBattle.trimp.gooStored -= dmg;
                    autoBattle.damageCreature(autoBattle.trimp, dmg, true);
                }
            },
            dustType: "shards",
            startPrice: 2500,
            priceMod: 15
        },
        Omni_Enhancer: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 210,
            description: function(){
                return "+" + prettify(this.attack()) + " Attack, +" + prettify(this.bleedMod() * 100) + "% Bleed Damage, +" + prettify(this.shockMod() * 100) + "% Shock Damage, +" + prettify(this.poisonMod()) + " Poison Damage, +1 Poison Stack Rate, and Poisons tick 10% faster.";
            },
            upgrade: "+2500 Attack, +300% Bleed Damage, +300% Shock Damage, +1000 Poison Damage",
            attack: function(){
                return 2500 + (2500 * this.level);
            },
            bleedMod: function(){
                return 4 + (3 * this.level);
            },
            shockMod: function(){
                return 4 + (3 * this.level);
            },
            poisonMod: function(){
                return (1000 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.bleedMod += this.bleedMod();
                autoBattle.trimp.shockMod += this.shockMod();
                autoBattle.trimp.poisonMod += this.poisonMod();
                autoBattle.trimp.poisonTick *= 0.9;
                autoBattle.trimp.poisonRate++;
            },
            dustType: "shards",
            startPrice: 6300,
            priceMod: 15
        },
        //stormbringer 215
        Box_of_Spores: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 220,
            description: function(){
            return "If the Enemy dies while Poisoned after never Bleeding, it drops " + this.dustMult() + "x more Dust.";
            },
            upgrade: "+1x Dust",
            dustMult: function(){
                return 4 + this.level;
            },
            dustType: "shards",
            startPrice: 60000,
            priceMod: 15
        },
        //nullifium armor 225
        //Myco Mitts 230
        Haunted_Harpoon: { 
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 235,
            description: function(){
                return "+" + prettify(this.health()) + " Health. If the Enemy is Bleeding and has been alive for at least 5 seconds, Huffy gains " + prettify(this.attack()) + " Attack, and the Enemy takes an additional " + prettify(this.bleedTickMult() * 100) + "% of its Bleed damage every second."
            },
            upgrade: "+10,000 Attack, +5000 Health, +100% of Bleed Damage taken per second",
            health: function(){
                return 5000 + (5000 * this.level);
            },
            bleedTickMult: function(){
                return 9 + this.level;
            },
            attack: function(){
                return 15000 + (10000 * this.level);
            },
            doStuff: function(){
                if (autoBattle.enemy.bleed.time > 0 && autoBattle.battleTime > 5000) autoBattle.trimp.attack += this.attack();
                autoBattle.trimp.maxHealth += this.health();
            },
            afterCheck: function(){
                if (autoBattle.enemy.bleed.time > 0 && autoBattle.battleTime > 5000){
                    var bdamage = autoBattle.getBleedDamage(autoBattle.enemy, autoBattle.trimp);
                    var pct = this.bleedTickMult() * (autoBattle.frameTime / 1000);
                    bdamage *= pct;
                    autoBattle.damageCreature(autoBattle.enemy, bdamage);
                    
                }
            },
            dustType: "shards",
            startPrice: 15e5,
            priceMod: 20
        },
        //Dopp Diadem 250
        //Final calc items
        //After all shock resist
        Stormbringer: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 215,
            description: function(){
                return "The Enemy loses all Shock Resistance, adding it instead to Bleed Resistance. +" + prettify(this.shockMod() * 100) + "% Shock Damage, +" + prettify(this.poisonMod()) + " Poison Damage.";
            },
            upgrade: "+500% Shock Damage, +5000 Poison Damage",
            shockMod: function(){
                return 10 + (5 * this.level);
            },
            poisonMod: function(){
                return 5000 + (5000 * this.level);
            },
            doStuff: function(){
                autoBattle.enemy.bleedResist += autoBattle.enemy.shockResist;
                autoBattle.enemy.shockResist = 0;
                autoBattle.trimp.shockMod += this.shockMod();
                autoBattle.trimp.poisonMod += this.poisonMod();
            },
            dustType: "shards",
            startPrice: 20000,
            priceMod: 15
        },
        Bad_Medkit: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 84,
            description: function(){
                return "Causes Bleeds you generate from other items to last at least " + this.bleedTime() + " seconds. +" + prettify(this.bleedChance()) + "% Bleed Chance. +" + prettify(this.lifesteal() * 100) + "% Lifesteal if the enemy is Bleeding.";
            },
            upgrade: "+1s Minimum Bleed Time, +4% Bleed Chance, +2.5% Lifesteal",
            bleedTime: function(){
                return 11 + (1 * this.level);
            },
            lifesteal: function(){
                return 0.175 + (0.025 * this.level);
            },
            bleedChance: function(){
                return 21 + (this.level * 4);
            },
            doStuff: function(){
                if (autoBattle.trimp.bleedTime > 0 && autoBattle.trimp.bleedTime < (this.bleedTime() * 1000)) autoBattle.trimp.bleedTime = this.bleedTime() * 1000;
                if (autoBattle.enemy.bleed.time > 0) autoBattle.trimp.lifesteal += this.lifesteal();
                autoBattle.trimp.bleedChance += this.bleedChance();
            },
            startPrice: 300
        },
        Putrid_Pouch: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 78,
            description: function(){
                return "-10% Attack Time and +" + prettify(this.defense()) + " Defense if the Enemy is Poisoned. Causes Poisons you generate from other items to last at least " + prettify(this.poisonTime() / 1000) + " seconds. +" + prettify(this.poisonChance()) + "% Poison Chance.";
            },
            upgrade: "+1s Poison Duration, +6% Poison Chance, +3 Defense",
            poisonTime: function(){
                return 19000 + (this.level * 1000);
            },
            poisonChance: function(){
                return 14 + (this.level * 6);
            },
            defense: function(){
                return 7 + (3 * this.level);
            },
            doStuff: function(){
                if (autoBattle.enemy.poison.time > 0){
                    autoBattle.trimp.attackSpeed *= 0.9;
                    autoBattle.trimp.defense += this.defense();
                }
                var poisonTime = this.poisonTime();
                if (autoBattle.trimp.poisonTime > 0 && autoBattle.trimp.poisonTime < poisonTime) autoBattle.trimp.poisonTime = poisonTime;
                autoBattle.trimp.poisonChance += this.poisonChance();
            },
            startPrice: 150,
            priceMod: 4
        },
        Raincoat: { //After all bleed chance
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 75,
            description: function(){
                return "If you have a chance to cause Bleeding, gain +" + prettify(this.defense()) + " Defense, +" + prettify(this.health()) + " Health, +" + prettify(this.lifesteal() * 100) + "% Lifesteal, and +" + prettify(this.bleedDamage() * 100) + "% Bleed Damage.";
            },
            upgrade: "+2 defense, +20 health, +2.5% Lifesteal, +10% Bleed Damage",
            defense: function(){
                return 4 + (this.level * 2);
            },
            health: function(){
                return 20 + (20 * this.level);
            },
            lifesteal: function(){
                return 0.125 + (0.025 * this.level)
            },
            bleedDamage: function(){
                return 0.2 + (0.1 * this.level);
            },
            doStuff: function(){
                var bleedChance = autoBattle.trimp.bleedChance;
                if (autoBattle.items.Sacrificial_Shank.equipped) bleedChance = Math.floor(bleedChance * 0.75);
                if (bleedChance > autoBattle.enemy.bleedResist && (autoBattle.trimp.bleedTime > 0 || autoBattle.items.Doppelganger_Diadem.equipped) && autoBattle.trimp.bleedMod > 0){
                    autoBattle.trimp.defense += this.defense();
                    autoBattle.trimp.maxHealth += this.health();
                    autoBattle.trimp.lifesteal += this.lifesteal();
                    autoBattle.trimp.bleedMod += this.bleedDamage();
                }
            },
            startPrice: 100,
            priceMod: 4
        },
        Labcoat: { //after all poison chance
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 90,
            description: function(){
                return "If you have a chance to cause Poison, gain +" + prettify(this.health()) + " Health, -" + prettify((1 - this.attackTime()) * 100) + "% Attack Time, and +" + prettify(this.poisonMod()) + " Poison Damage.";
            },
            upgrade: "+25 Health, -1% Attack Time, +1 Poison Damage",
            health: function(){
                return 25 + (25 * this.level);
            },
            attackTime: function(){
                return Math.pow(0.99, this.level);
            },
            poisonMod: function(){
                return 1 + this.level;
            },
            doStuff: function(){
                var poisonChance = autoBattle.trimp.poisonChance;
                if (autoBattle.items.Sacrificial_Shank.equipped) poisonChance = Math.floor(poisonChance * 0.75);
                if (poisonChance > autoBattle.enemy.poisonResist && autoBattle.trimp.poisonMod > 0 && autoBattle.trimp.poisonTime > 0){
                    autoBattle.trimp.maxHealth += this.health();
                    autoBattle.trimp.attackSpeed *= this.attackTime();
                    autoBattle.trimp.poisonMod += this.poisonMod();
                }
            },
            startPrice: 1500,
            priceMod: 4.5
        },
        Aegis: { //after all health
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 118,
            description: function(){
                return "+" + this.defenseEffect() + " Defense. If Huffy's Health % is higher than Enemy Health %, gain +" + prettify(this.shockChance()) + "% Shock Chance, +" + prettify(this.shockMod() * 100) + "% Shock Damage, 15s Shock Time. Otherwise, this item's Defense is doubled and gain +" + prettify(this.lifestealEffect() * 100) + "% Lifesteal.";
            },
            upgrade: "+4 Defense, +10% Shock Chance, +10% Shock Damage, +10% Lifesteal",
            defenseEffect: function(){
                return 6 + (4 * this.level);
            },
            shockChance: function(){
                return 15 + (10 * this.level);
            },
            shockMod: function(){
                return 0.15 + (0.1 * this.level);
            },
            lifestealEffect: function(){
                return 0.05 + (0.1 * this.level);
            },
            doStuff: function(){
                var hufPct = autoBattle.trimp.health / autoBattle.trimp.maxHealth;
                var enemyPct = autoBattle.enemy.health / autoBattle.enemy.maxHealth;
                if (hufPct > enemyPct){
                    autoBattle.trimp.shockChance += this.shockChance();
                    autoBattle.trimp.shockMod += this.shockMod();
                    if (autoBattle.trimp.shockTime < 15000) autoBattle.trimp.shockTime = 15000;
                    autoBattle.trimp.defense += this.defenseEffect();
                }
                else{
                    autoBattle.trimp.lifesteal += this.lifestealEffect();
                    autoBattle.trimp.defense += (this.defenseEffect() * 2);
                }
            },
            priceMod: 8,
            startPrice: 65000,
        },
        Sacrificial_Shank: { //after all status chances
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 145,
            enemyReduced: 0,
            description: function(){
                return "Multiply Huffy's and the Enemy's highest status effect chance (before resists) by 0.75. -" + prettify((1 - this.attackTime()) * 100) + "% Attack Time, +" + prettify(this.resist()) + "% to all Resists, and +" + prettify(this.lifesteal() * 100) + "% Lifesteal per 10% Huffy or Enemy status chance lost.";
            },
            upgrade: "-1% Attack Time, +1% Resists, +1% Lifesteal per 10% status chance lost",
            attackTime: function(){
                return Math.pow(0.99, this.level);
            },
            resist: function(){
                return (1 * this.level);
            },
            lifesteal: function(){
                return (0.01 * this.level);
            },
            onEnemy: function(){
                var toReduce = ["poisonChance", "bleedChance", "shockChance"];
                var totalReduce = 0;
                var highestElem = "";
                var highestChance = 0;
                for (var x = 0; x < toReduce.length; x++){
                    var name = toReduce[x];
                    if (autoBattle.enemy[name] > highestChance) {
                        highestChance = autoBattle.enemy[name];
                        highestElem = name;
                    }
                }
                var thisReduce = autoBattle.enemy[highestElem] * 0.25;
                if (thisReduce > 0){
                    autoBattle.enemy[highestElem] -= thisReduce;
                    totalReduce += thisReduce;
                }
                this.enemyReduced = totalReduce;
            },
            doStuff: function(){
                var toReduce = ["poisonChance", "bleedChance", "shockChance"];
                var totalReduce = this.enemyReduced;
                var highestElem = "";
                var highestChance = 0;
                for (var x = 0; x < toReduce.length; x++){
                    var name = toReduce[x];
                    if (autoBattle.trimp[name] > highestChance) {
                        highestChance = autoBattle.trimp[name];
                        highestElem = name;
                    }
                }
                var thisReduce = autoBattle.trimp[highestElem] * 0.25;
                if (thisReduce > 0){
                    autoBattle.trimp[highestElem] -= thisReduce;
                    totalReduce += thisReduce;
                }
                totalReduce = Math.floor(totalReduce / 10);
                if (totalReduce <= 0) return;
                autoBattle.trimp.attackSpeed *= Math.pow(this.attackTime(), totalReduce);
                autoBattle.trimp.lifesteal += (this.lifesteal() * totalReduce);
                autoBattle.trimp.poisonResist += (this.resist() * totalReduce);
                autoBattle.trimp.bleedResist += (this.resist() * totalReduce);
                autoBattle.trimp.shockResist += (this.resist() * totalReduce);
            },
            startPrice: 2500000,
            priceMod: 4
        },
        Basket_of_Souls: { //after all additive lifesteal and health (before monkimp)
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 200,
            description: function(){
                return "+" + prettify(this.health()) + " Health, +" + prettify(this.defense()) + " Defense, +" + prettify(this.lifesteal() * 100) + "% Lifesteal, +" + prettify(this.bleedMod() * 100) + "% Bleed Damage, +" + prettify(this.shockMod() * 100) + "% Shock Damage. Multiplies Huffy's Max Health by his Lifesteal value, then multiplies his Lifesteal by 0.5."
            },
            upgrade: "+1000 Health, +300 Defense, +100% Lifesteal, +200% Bleed Damage, +200% Shock Damage",
            health: function(){
                return 2000 + (1000 * this.level);
            },
            defense: function(){
                return 300 + (200 * this.level);
            },
            lifesteal: function(){
                return 2 + (this.level);
            },
            bleedMod: function(){
                return 3 + (2 * this.level);
            },
            shockMod: function(){
                return 3 + (2 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.maxHealth += this.health();
                autoBattle.trimp.defense += this.defense();
                autoBattle.trimp.lifesteal += this.lifesteal();
                autoBattle.trimp.bleedMod += this.bleedMod();
                autoBattle.trimp.shockMod += this.shockMod();
                if (autoBattle.items.Monkimp_Paw.equipped) autoBattle.trimp.lifesteal *= 0.75; //monkimp paw special interaction
                autoBattle.trimp.maxHealth *= autoBattle.trimp.lifesteal;
                autoBattle.trimp.lifesteal *= 0.5;
            },
            dustType: "shards",
            startPrice: 1000,
            priceMod: 15
            
        },
        Monkimp_Paw: { //after basket of souls
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 155,
            description: function(){
                return "+" + prettify(this.attack()) + " Attack, removes a fourth of your total Lifesteal."
            },
            upgrade: "+100 Attack",
            attack: function(){
                return (100 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.attack += this.attack();
                if (!autoBattle.items.Basket_of_Souls.equipped) autoBattle.trimp.lifesteal *= 0.75; //basket of souls special interaction
                if (autoBattle.trimp.lifesteal < 0) autoBattle.trimp.lifesteal = 0;
            },
            startPrice: 200e6,
            priceMod: 10
        },
        Spiked_Gloves: { //after all attack
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 108,
            description: function(){
                return "+" + this.formatEffect() + "% Attack damage.";
            },
            upgrade: "+5% attack damage",
            formatEffect: function(){
                return prettify(this.effect() * 100);
            },
            effect: function(){
                return .2 + (0.05 * this.level);
            },
            doStuff: function(){
                autoBattle.trimp.attack *= (1 + this.effect());
            },
            startPrice: 10000,
            priceMod: 6,
        },
        //after all attack and health
        Nullifium_Armor: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 225,
            description: function(){
            return "Huffy gains +" + prettify((this.statMult() -1) * 100) + "% Health, and +" + prettify((this.statMult() -1) * 100) + "% Attack. If the Enemy is not Poisoned, gain " + prettify(this.lifesteal() * 100) + "% Lifesteal.";
            },
            upgrade: "+50% Attack, Health, and Lifesteal",
            statMult: function(){
                return 4.5 + (this.level * 0.5);
            },
            lifesteal: function(){
                return 1.5 + (this.level * 0.5);
            },
            doStuff: function(){
                if (autoBattle.enemy.poison.time <= 0) autoBattle.trimp.lifesteal += this.lifesteal();
                autoBattle.trimp.maxHealth *= this.statMult();
                autoBattle.trimp.attack *= this.statMult();
            },
            dustType: "shards",
            startPrice: 200000,
            priceMod: 20
        },
        Doppelganger_Diadem: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 250,
            description: function(){
                return "Can create a Bleed on the Enemy for 10 seconds. 2x Attack if the Signet is equipped, x2 again if Doppelganger is alive. Doppelganger has 50% more health and can resurrect once per fight.";
            },
            doStuff: function(){
                if (autoBattle.trimp.bleedTime <= 10000) autoBattle.trimp.bleedTime = 10000;
                if (autoBattle.items.Doppelganger_Signet.equipped){
                    autoBattle.trimp.attack *= 2;
                    if (!autoBattle.trimp.doppDown) autoBattle.trimp.attack *= 2;
                    if (typeof autoBattle.trimp.doppLives === 'undefined') autoBattle.trimp.doppLives = 1;
                }

            },
            noUpgrade: true,
            dustType: "shards",
        },
        Doppelganger_Signet: { //actual final attack item
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 190,
            longText: true,
            description: function(){
                return "Summon a Doppelganger which grants you 50% damage reduction, 2x Attack, and +1 Poison Stack Rate while it is alive. Your Doppelganger will explode after taking damage equal to your Max Health or if it would kill the Enemy, redealing all damage dealt so far this fight, and shredding 50% Enemy Defense.";
            },
            onDeath: function(){
                var damageDealt = autoBattle.enemy.dmgTaken;
                autoBattle.damageCreature(autoBattle.enemy, damageDealt, false, true);
                autoBattle.enemy.defense *= 0.5;
                if (autoBattle.trimp.doppLives) autoBattle.trimp.doppLives--;
                else autoBattle.trimp.doppDown = true;
            },
            doppHealth: function(){
                var healthAmt = autoBattle.trimp.maxHealth;
                if (autoBattle.items.Doppelganger_Diadem.equipped){
                    healthAmt *= 1.5;
                    if (!autoBattle.trimp.doppLives) healthAmt *= 2;
                }
                return healthAmt;
            },
            explodeDmg: function(){
                var damageAmt = autoBattle.enemy.dmgTaken;
                if (autoBattle.items.Doppelganger_Diadem.equipped && autoBattle.trimp.doppLives == 1) damageAmt *= 3;
                return damageAmt;
            },
            doStuff: function(){
                if (autoBattle.trimp.doppDown) return;
                autoBattle.trimp.attack *= 2;
                autoBattle.trimp.damageTakenMult *= 0.5;
                autoBattle.trimp.poisonRate++;
                if (autoBattle.trimp.dmgTaken >= this.doppHealth() || this.explodeDmg() >= autoBattle.enemy.health) this.onDeath();
            },
            noUpgrade: true,
            dustType: "shards"
        },
        //Final Poison Damage item
        Myco_Mitts: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 230,
            description: function(){
                return "Multiplies the damage dealt by Poison ticks by " + this.poisonMult() + "x."
            },
            upgrade: "+1x to Poison Damage multiplier",
            poisonMult: function(){
                return 2 + this.level;
            },
            doStuff: function(){
                autoBattle.trimp.poisonMod *= this.poisonMult();
            },
            startPrice: 5e5,
            priceMod: 20,
            dustType: "shards",
        },
        Gaseous_Greataxe: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 260,
            dustMult: function(){
                var mult = 0.75 + ((this.level - 1) * 0.01);
                if (mult > 1) mult = 1;
                return mult; 
            },
            description: function(){
                return "Multiplies your Poison Damage by your Bleed Damage or Shock Damage, whichever is higher. Doubles Poison Stack Rate. +" + prettify(this.maxStacks()) + " Max Poison Stacks. Every 10 seconds increases Max Poison stacks by 10% of this amount and increases Poison damage by 10% (compounding). Reduces Dust and Shard gains by " + Math.round((1 - this.dustMult()) * 100) + "%.";
            },
            upgrade: "+20% more (compounding) Max Poison Stacks, -1% to Dust Penalty",
            maxStacks: function(){
                return Math.floor(100 * Math.pow(1.2, this.level - 1));
            },
            doStuff: function(){
                var higherMod = (autoBattle.trimp.shockMod > autoBattle.trimp.bleedMod) ? autoBattle.trimp.shockMod : autoBattle.trimp.bleedMod;
                if (higherMod < 1) higherMod = 1;
                autoBattle.trimp.poisonMod *= higherMod;
                var timeStacks = Math.floor(autoBattle.battleTime / 10000);
                autoBattle.trimp.poisonMod *= Math.pow(1.1, timeStacks);
                autoBattle.trimp.poisonStack += Math.floor(this.maxStacks() * ((timeStacks + 10) * 0.1));
                autoBattle.trimp.poisonRate *= 2;
            },
            dustType: "shards",
            startPrice: 1e10,
            priceMod: 20
        },
        The_Fibrillator: {
            owned: false,
            equipped: false,
            hidden: false,
            level: 1,
            zone: 270,
            description: function(){
                return "Multiplies Attack and Poison Damage by " + prettify(this.shockMod()) + "x. Adds " + prettify(this.shockMod() - 1) + " to this item's Attack and Poison Damage multplier for every 10 seconds the current shock on the enemy has lasted. Shocks last a minimum of 30 seconds (Overrides Sundering Scythe). Multiplies Health by " + prettify(this.healthMod()) + "x";
            },
            upgrade: "+0.05 to scaling Attack and Poison Multiplier, +0.25x Health",
            shockMod: function(){
                return 1.1 + ((this.level - 1) * 0.05);
            },
            healthMod: function(){
                return 2 + ((this.level - 1) * 0.25);
            },
            doStuff: function(){
                autoBattle.trimp.maxHealth *= this.healthMod();
                var shockSeconds = 0;
                if (autoBattle.enemy.shock.time > 0){
                    shockSeconds = (autoBattle.battleTime - autoBattle.enemy.shock.timeApplied) / 1000;
                    shockSeconds = Math.floor(shockSeconds / 10);
                    var shockMod = this.shockMod();
                    var toAdd = (shockMod - 1) * shockSeconds;
                    toAdd += shockMod;
                    autoBattle.trimp.attack *= toAdd;
                    autoBattle.trimp.poisonMod *= toAdd;
                }
            },
            dustType: "shards",
            startPrice: 1e12,
            priceMod: 20
        }

    },
    bonuses: {
        Extra_Limbs: {
            description: function(){
                return "Huffy can equip 1 additional item.<br/><br/>"
            },
            level: 0,
            price: 100,
            priceMod: 100
        },
        Radon: {
            description: function(){
                return "Increase all Radon earned by +10% per level.<br/>"
            },
            getMult: function(){
                return 1 + (this.level * 0.1);
            },
            level: 0,
            price: 30000,
            priceMod: 3,
            max: 50
        },
        Stats: {
            description: function(){
                return "Increases Attack and Health in U2 by +10% per level.<br/>"
            },
            getMult: function(){
                return 1 + (this.level * 0.1);
            },
            level: 0,
            price: 20000,
            priceMod: 3,
            max: 50
        },
        Scaffolding: {
            description: function(){
                return "Each level adds +100% Housing and increases the bonus of all other Scaffolds by 10%."
            },
            getMult: function(){
                return 1 + (this.level * Math.pow(1.1, this.level - 1))
            },
            level: 0,
            price: 50,
            useShards: true,
            priceMod: 10,
            max: 18
        }
    },
    oneTimers: {
        Gathermate: {
            get description(){
                return "Gather 5% more Food, Wood, and Metal in U2 for each Spire Assault level cleared.";
            },
            owned: false,
            getMult: function(){
                return 1 + (0.05 * (autoBattle.maxEnemyLevel - 1));
            },
            requiredItems: 7
        },
        Smithriffic: {
            description: "Get an extra Smithy when completing Melting Point.",
            owned: false,
            requiredItems: 11
        },
        Championism: {
            description: "Unlock a new Perk!",
            owned: false,
            onPurchase: function(){
                game.portal.Championism.radLocked = false;
            },
            requiredItems: 15
        },
        Master_of_Arms: {
            description: "Huffy gains +200 Health, +10 Attack, and +2 Poison Damage.",
            owned: false,
            requiredItems: 19
        },
        Artisan: {
            get description(){
                return "All U2 Equipment costs 1% less for each SA level cleared. (Currently " + prettify((1 - this.getMult()) * 100) + "% cheaper)";
            },
            owned: false,
            getMult: function(){
                return Math.pow(0.99, autoBattle.maxEnemyLevel - 1);
            },
            requiredItems: 21
        },
        Battlescruff: {
            description: "Increases all Scruffy XP gained by +2% for each Spire Assault level cleared.",
            owned: false,
            requiredItems: 23
        },
        Collectology: {
            description: "Collectors add 2 Hubs each PLUS another extra Hub for every 30 Spire Assault levels cleared.",
            owned: false,
            requiredItems: 28,
            getHubs: function(){
                return 2 + Math.floor((autoBattle.maxEnemyLevel - 1) / 30);
            }
        },
        Dusty_Tome: {
            description: "+5% Dust found on all levels per Spire Assault level cleared.",
            owned: false,
            requiredItems: 32
        },
        Whirlwind_of_Arms: {
            description: "+1000 Health, +25 Attack, +10 Poison Damage.",
            owned: false,
            requiredItems: 34
        },
        Nullicious: {
            description: "Increase the base Nu value of U2 Heirlooms by 0.5% per Spire Assault level cleared.",
            owned: false,
            requiredItems: 36,
            getMult: function(){
                return 1 + ((autoBattle.maxEnemyLevel - 1) * 0.005);
            }
        },
        Suprism: {
            description: "Increases Prismatic Shield by 3% per Spire Assault level cleared.",
            getMult: function(){
                return ((autoBattle.maxEnemyLevel -1) * 0.03);
            },
            owned: false,
            requiredItems: 39,
        },
        The_Ring: {
            description: "Unlock The Ring.",
            owned: false,
            requiredItems: 42,
            useShards: true,
            onPurchase: function(){
                document.getElementById('autoBattleRingBtn').style.display = 'inline-block';
            }
        },
        Mass_Hysteria: {
            description: "Frenzy is always active.",
            owned: false,
            requiredItems: 45,
            useShards: true
        },
        Burstier: {
            description: "Gamma Burst requires 1 fewer attack before triggering.",
            owned: false,
            requiredItems: 48,
            useShards: true
        },
        Expanding_Tauntimp: {
            description: "Starting after your next Portal, U2 Tauntimps will increase all Trimps gained by " + prettify(game.badGuys.Tauntimp.expandingBase() * 100) + "% per run instead of adding flat housing.",
            owned: false,
            requiredItems: 51,
            useShards: true
        },
        More_Expansion: {
            description: "Unlock the Expansion Perk, further increasing the power of your Tauntimps",
            owned: false,
            requiredItems: 53,
            owned: false,
            useShards: true,
            onPurchase: function(){
                game.portal.Expansion.radLocked = false;
            }
        }
    },
    fight: function(){
        if (!this.trimp || !this.enemy) this.resetCombat();
        this.enemy.lastAttack += this.frameTime;
        this.trimp.lastAttack += this.frameTime;

        this.enemy.maxHealth = this.enemy.baseHealth;
        this.trimp.maxHealth = this.trimp.baseHealth;
        this.enemy.attackSpeed = this.enemy.baseAttackSpeed;
        this.trimp.attackSpeed = this.trimp.baseAttackSpeed;
        this.trimp.attack = this.trimp.baseAttack;
        this.enemy.attack = this.enemy.baseAttack;

        this.trimp.shockChance = 0;
        this.trimp.shockMod = 0;
        this.trimp.shockTime = 0;
        
        this.trimp.bleedChance = 0;
        this.trimp.bleedMod = 0;
        this.trimp.bleedTime = 0;

        this.trimp.poisonChance = 0;
        this.trimp.poisonTime = 0;
        this.trimp.poisonMod = 0;
        this.trimp.poisonStack = 2;
        this.trimp.poisonTick = 1000;
        this.trimp.poisonHeal = 0;
        this.trimp.poisonRate = 1;

        this.trimp.shockResist = 0;
        this.trimp.poisonResist = 0;
        this.trimp.bleedResist = 0;

        this.trimp.defense = 0;
        this.trimp.lifesteal = 0;
        this.trimp.damageTakenMult = 1;
        this.trimp.slowAura = 1;

        this.trimp.dustMult = 0;
        
        this.checkItems();

        if (this.enemy.immune != ""){
            this.trimp[this.enemy.immune + "Chance"] = 0;
        }
        var trimpAttackTime = this.trimp.attackSpeed;
        if (this.trimp.lastAttack >= trimpAttackTime){
            this.trimp.lastAttack -= trimpAttackTime;
            this.attack(this.trimp, this.enemy);
        }
        this.checkPoison(this.trimp);
        if (this.trimp.bleed.time > 0) this.trimp.bleed.time -= this.frameTime;
        if (this.trimp.shock.time > 0) this.trimp.shock.time -= this.frameTime;
        if (this.enemy.health <= 0) {
            this.enemyDied();
            return;
        }
        if (this.trimp.health <= 0){
            this.trimpDied();
            return;
        }
        if (!this.enemy.noSlow) this.enemy.attackSpeed *= this.trimp.slowAura;
        var enemyAttackTime = this.enemy.attackSpeed;
        if (this.enemy.lastAttack >= enemyAttackTime){
            this.enemy.lastAttack -= enemyAttackTime;
            this.attack(this.enemy, this.trimp);
        }
        if (this.enemy.explodeFreq != -1){
            this.enemy.lastExplode += this.frameTime;
            if (this.enemy.lastExplode >= this.enemy.explodeFreq){
                this.enemy.lastExplode -= this.enemy.explodeFreq;
                var dmg = (this.enemy.explodeDamage * this.getAttack(this.enemy)) - this.trimp.defense;
                this.damageCreature(this.trimp, dmg);
            }
        }
        this.checkPoison(this.enemy);
        if (this.enemy.bleed.time > 0) this.enemy.bleed.time -= this.frameTime;
        if (this.enemy.shock.time > 0 && this.enemy.shock.time != 9999999) this.enemy.shock.time -= this.frameTime;
        if (this.trimp.health > this.trimp.maxHealth) this.trimp.health = this.trimp.maxHealth;
        if (this.enemy.health > this.enemy.maxHealth) this.enemy.health = this.enemy.maxHealth;
        if (this.trimp.health <= 0){
            this.trimpDied();
            return;
        }
        if (this.enemy.health <= 0) {
            this.enemyDied();
            return;
        }
    },
    checkItems: function(){
        if (this.oneTimers.Master_of_Arms.owned){
            this.trimp.maxHealth += 200;
            this.trimp.attack += 10;
            this.trimp.poisonMod += 2;
        }
        if (this.oneTimers.Whirlwind_of_Arms.owned){
            this.trimp.maxHealth += 1000;
            this.trimp.attack += 25;
            this.trimp.poisonMod += 10;
        }
        for (var x = 0; x < this.rings.mods.length; x++){
            var modObj = this.ringStats[this.rings.mods[x]];
            modObj.doStuff(this.getRingStatAmt(modObj));
        }
        var ringDmg = this.getRingStatusDamage();
        if (ringDmg > 0){
            ringDmg /= 100;
            this.trimp.bleedMod += ringDmg;
            this.trimp.shockMod += ringDmg;
            this.trimp.poisonMod += this.getRingPoisonDamage();
        }
        var ringChance = this.getRingStatusChance();
        if (ringChance > 0){
            this.trimp.bleedChance += ringChance;
            this.trimp.shockChance += ringChance;
            this.trimp.poisonChance += ringChance;
        }
        if (this._equippedItems) {
            for (var ei = 0, elen = this._equippedItems.length; ei < elen; ei++) {
                this._equippedItems[ei].doStuff();
            }
        } else {
            for (var item in this.items){
                var itemObj = this.items[item];
                if (!itemObj.equipped) continue;
                if (itemObj.doStuff) itemObj.doStuff();
            }
        }

        if (this.items.Sundering_Scythe.equipped && this.trimp.shockTime > 10000) this.trimp.shockTime = 10000;
        if (this.items.The_Fibrillator.equipped && this.trimp.shockTime > 0 && this.trimp.shockTime < 30000) this.trimp.shockTime = 30000;
        if (this.items.Blessed_Protector.equipped) this.items.Blessed_Protector.afterCheck(); //after anything that might hurt huffy
        if (this.items.Grounded_Crown.equipped) this.items.Grounded_Crown.afterCheck(); //just deals damage
        if (this.items.Haunted_Harpoon.equipped) this.items.Haunted_Harpoon.afterCheck(); //after anything that might increase the damage of huffy's bleeds
        
        this.trimp.attackSpeed *= this.enemy.slowAura;
        if (this.trimp.attackSpeed <= 500){
            this.trimp.slowAura += ((500 - this.trimp.attackSpeed) / 1000)
            this.trimp.attackSpeed = 500;
        }
    },
    damageCreature: function(creature, dmg, fromGoo, ignoreEth){
        dmg *= creature.damageTakenMult;
        if (creature.isEthereal && !ignoreEth) creature.health += dmg;
        else {
            if (!fromGoo && creature.isTrimp && this.items.Goo_Golem.equipped && this.items.Goo_Golem.active()){
                creature.gooStored += dmg;
            }
            else {
                creature.health -= dmg;
                creature.dmgTaken += dmg;
            }
        }
        return dmg;
    },
    checkPoison: function(creature){
        var opponent = creature.isTrimp ? this.enemy : this.trimp;
        if (creature.poison.time > 0){
            creature.poison.lastTick += this.frameTime;
            var tickTime = opponent.poisonTick;
            if (creature.poison.lastTick >= tickTime){
                var shockMod = 1;
                if (creature.shock.time > 0){
                    shockMod += creature.shock.mod;
                }
                creature.poison.lastTick -= tickTime;
                creature.poison.time -= tickTime;
                var dmg = (creature.poison.mod * creature.poison.stacks * shockMod)
                dmg = this.damageCreature(creature, dmg);
                if (opponent.poisonHeal) {
                    var healFor = (opponent.poisonHeal * creature.poison.stacks);
                    if (this.items.The_Globulator.equipped) healFor *= 2;
                    this.trimp.health += healFor;
                    if (this.trimp.health > this.trimp.maxHealth) this.trimp.health = this.trimp.maxHealth;
                }
                if (creature.poison.time <= 0){
                    creature.poison.time = 0;
                    creature.poison.mod = 0;
                    creature.poison.lastTick = 0;
                    creature.poison.stacks = 0;
                    creature.poison.expired = true;
                    creature.poison.hitsAtMax = 0;
                }
            }
        }
    },
    getAttack: function(fighterObj){
        if (fighterObj.isTrimp) return fighterObj.attack;
        return fighterObj.attack * this.getEnrageMult() * this.getBerserkMult();
    },
    getBerserkMult: function(){
        if (this.enemy.berserkMod == -1) return 1;
        return Math.pow(this.enemy.berserkMod, Math.floor(this.enemy.berserkStack / this.enemy.berserkEvery));
    },
    rollDamage: function(attacker){
        var baseAttack = this.getAttack(attacker);
        var attack = baseAttack * 0.2;
        var roll = Math.floor(Math.random() * 201);
        roll -= 100;
        roll /= 100;
        return (baseAttack + (attack * roll));
    },
    attack: function(attacker, defender){
        var damage = this.rollDamage(attacker);
        var shockMod = 1;
        if (defender.shock.time > 0){
            shockMod = (1 + defender.shock.mod);
            damage *= shockMod;
        }
        damage -= defender.defense;
        if (damage < 0) damage = 0;
        damage = this.damageCreature(defender, damage);
        var atkLifesteal = attacker.lifesteal - defender.lifestealResist;
        if (atkLifesteal > 0){
            attacker.health += (damage * atkLifesteal);
            if (attacker.health > attacker.maxHealth) attacker.health = attacker.maxHealth;
        }
        if (attacker.bleed.time > 0){
            var bdamage = this.getBleedDamage(attacker, defender);
            bdamage = this.damageCreature(attacker, bdamage);
            var defLifesteal = defender.lifesteal - attacker.lifestealResist;
            if (defLifesteal > 0){
                var healAmt = (bdamage * defLifesteal);
                if (defender.isTrimp && this.items.Recycler.equipped) healAmt *= 2;
                defender.health += healAmt;
                if (defender.health > defender.maxHealth) defender.health = defender.maxHealth;
            }
            if (attacker.bleed.time <= 0){
                attacker.bleed.time = 0;
                attacker.bleed.mod = 0;
            } 
        }
        var bleedChance = attacker.bleedChance - defender.bleedResist;
        if (bleedChance > 0 && attacker.bleedMod > 0 && attacker.bleedTime > 0 && (defender.bleed.time <= 0 || this.items.Sundering_Scythe.equipped && defender.bleed.time <= 5000)){
            var roll = Math.floor(Math.random() * 100);
            if (roll < bleedChance){
                if (this.items.Bloodstained_Gloves.equipped) this.items.Bloodstained_Gloves.onBleed();
                if (this.items.Bag_of_Nails.equipped) this.enemy.noSlow = true;
                if (defender.bleed.mod < attacker.bleedMod) defender.bleed.mod = (1 + attacker.bleedMod);
                if (defender.bleed.time < attacker.bleedTime) defender.bleed.time = attacker.bleedTime;
                if (defender.bleed.time > 0) defender.hadBleed = true;
            }
        }
        var poisonChance = attacker.poisonChance - defender.poisonResist;
        if (poisonChance > 0 && attacker.poisonMod > 0 && attacker.poisonTime > 0){
            var roll = Math.floor(Math.random() * 100);
            if (roll < poisonChance){
                if (defender.poison.time < attacker.poisonTime) defender.poison.time = attacker.poisonTime;
                var stackRate = attacker.poisonRate;
                for (var x = 0; x < stackRate; x++){
                    defender.poison.mod = attacker.poisonMod;
                    if (defender.poison.stacks < attacker.poisonStack){
                        defender.poison.stacks++;
                        if (attacker.isTrimp && this.items.The_Globulator.equipped) this.items.The_Globulator.onPoisonStack(defender.poison.stacks);
                    }
                    else defender.poison.hitsAtMax++;
                }
            }
        }
        var shockChance = attacker.shockChance - defender.shockResist;
        if (shockChance > 0 && attacker.shockMod > 0 && attacker.shockTime > 0 && (defender.shock.time <= 0 || (defender.shock.time == 9999999 && attacker.shockMod > defender.shock.mod))){
            var roll = Math.floor(Math.random() * 100);
            if (roll < shockChance){
                if (attacker.isTrimp && this.items.Eelimp_in_a_Bottle.equipped) defender.lastAttack = 0;
                defender.shock.time = attacker.shockTime;
                defender.shock.timeApplied = autoBattle.battleTime;
                defender.shock.mod = attacker.shockMod;
                defender.shock.count++;
            }
        }
        if (attacker.berserkMod != -1) attacker.berserkStack++;
        if (attacker.ethChance > 0) {
            var ethRoll = Math.floor(Math.random() * 100);
            if (ethRoll < attacker.ethChance) attacker.isEthereal = true;
            else attacker.isEthereal = false;
        }
    },
    getBleedDamage: function(attacker, defender){
        var attackerShock = 1;
        if (attacker.shock.time > 0){
            attackerShock = (1 + attacker.shock.mod);
        }
        var bdamage = this.getAttack(defender) * attacker.bleed.mod * attackerShock;
        bdamage -= attacker.defense;
        return bdamage;
    },
    _enemyTemplateCache: sharedEnemyTemplateCache,
    resetCombat: function(resetStats){
        this.trimp = this.template();
        this.trimp.isTrimp = true;
        this.enemy = this.template();
        this.battleTime = 0;
        this.checkItems();
        this.trimp.health = this.trimp.maxHealth;
        var lvl = this.enemyLevel;
        var cacheKey = this.seed + ":" + lvl;
        var cached = this._enemyTemplateCache[cacheKey];
        if (!cached) {
            this.enemy.level = lvl;
            var atkSpdLevel = Math.min(lvl, 29);
            this.enemy.baseAttackSpeed *= Math.pow(0.98, atkSpdLevel);
            if (lvl >= 30){
                atkSpdLevel = lvl - 29;
                this.enemy.slowAura = Math.pow(1.01, atkSpdLevel);
            }
            this.enemy.baseHealth *= Math.pow(1.205, lvl);
            this.enemy.baseAttack += (2 * (lvl - 1));
            this.enemy.baseAttack *= Math.pow(1.04, lvl);
            if (lvl >= 50){
                var newLev = lvl - 49;
                this.enemy.baseHealth *= Math.pow(1.1, newLev);
                this.enemy.baseAttack *= Math.pow(1.1, newLev);
            }
            this.enemy.defense += (0.5 * lvl);
            this.enemy.poisonResist += lvl;
            this.enemy.bleedResist += lvl;
            this.enemy.shockResist += lvl;
            if (lvl >= 15) this.enemy.lifestealResist += (0.03 * (lvl - 14));
            if (lvl >= 30) this.enemy.enrageMult = 1.5;
            this.setProfile();
            cached = {
                profile: this.profile,
                level: lvl,
                baseAttackSpeed: this.enemy.baseAttackSpeed,
                slowAura: this.enemy.slowAura,
                baseHealth: this.enemy.baseHealth,
                baseAttack: this.enemy.baseAttack,
                defense: this.enemy.defense,
                poisonResist: this.enemy.poisonResist,
                bleedResist: this.enemy.bleedResist,
                shockResist: this.enemy.shockResist,
                lifestealResist: this.enemy.lifestealResist,
                enrageMult: this.enemy.enrageMult,
                enrageFreq: this.enemy.enrageFreq,
                poisonChance: this.enemy.poisonChance,
                poisonMod: this.enemy.poisonMod,
                poisonStack: this.enemy.poisonStack,
                poisonTime: this.enemy.poisonTime,
                bleedChance: this.enemy.bleedChance,
                bleedMod: this.enemy.bleedMod,
                bleedTime: this.enemy.bleedTime,
                shockChance: this.enemy.shockChance,
                shockMod: this.enemy.shockMod,
                shockTime: this.enemy.shockTime,
                lifesteal: this.enemy.lifesteal,
                immune: this.enemy.immune,
                explodeDamage: this.enemy.explodeDamage,
                explodeFreq: this.enemy.explodeFreq,
                berserkMod: this.enemy.berserkMod,
                berserkEvery: this.enemy.berserkEvery,
                ethChance: this.enemy.ethChance
            };
            this._enemyTemplateCache[cacheKey] = cached;
        } else {
            this.profile = cached.profile;
            var e = this.enemy;
            e.level = cached.level;
            e.baseAttackSpeed = cached.baseAttackSpeed;
            e.slowAura = cached.slowAura;
            e.baseHealth = cached.baseHealth;
            e.baseAttack = cached.baseAttack;
            e.defense = cached.defense;
            e.poisonResist = cached.poisonResist;
            e.bleedResist = cached.bleedResist;
            e.shockResist = cached.shockResist;
            e.lifestealResist = cached.lifestealResist;
            e.enrageMult = cached.enrageMult;
            e.enrageFreq = cached.enrageFreq;
            e.poisonChance = cached.poisonChance;
            e.poisonMod = cached.poisonMod;
            e.poisonStack = cached.poisonStack;
            e.poisonTime = cached.poisonTime;
            e.bleedChance = cached.bleedChance;
            e.bleedMod = cached.bleedMod;
            e.bleedTime = cached.bleedTime;
            e.shockChance = cached.shockChance;
            e.shockMod = cached.shockMod;
            e.shockTime = cached.shockTime;
            e.lifesteal = cached.lifesteal;
            e.immune = cached.immune;
            e.explodeDamage = cached.explodeDamage;
            e.explodeFreq = cached.explodeFreq;
            e.berserkMod = cached.berserkMod;
            e.berserkEvery = cached.berserkEvery;
            e.ethChance = cached.ethChance;
        }
        this.enemy.maxHealth = this.enemy.baseHealth;
        this.enemy.health = this.enemy.baseHealth;
        if (this.items.Fearsome_Piercer.equipped) this.items.Fearsome_Piercer.onEnemy();
        if (this.items.Sacrificial_Shank.equipped) this.items.Sacrificial_Shank.onEnemy();
        
        this.fight();
        if (resetStats) this.resetStats();
    },
    setProfile: function(){
        this.profile = "";
        if (this.enemyLevel == 1) return;
        var seed = this.seed;
        
        seed += (100 * this.enemyLevel);
        if (this.enemyLevel >= 51) seed += 3125; //Generated with Shold brain RNG
        var doubleResist = true;
        if (this.enemyLevel > 50){
            doubleResist = getRandomIntSeeded(seed++, 0, 100);
            doubleResist = (doubleResist < 20);
        }
        if (this.enemyLevel <= 50) doubleResist = true;
        var effects = ["Healthy", "Fast", "Strong", "Defensive"];
        if (this.enemyLevel > 5) {
            effects.push("Poisoning", "Bloodletting", "Shocking", "Lifestealing");
        }
        if (this.enemyLevel > 10){
            effects.push("Poison Resistant", "Shock Resistant", "Bleed Resistant");
        }
        if (this.enemyLevel > 20){
            effects.push("Enraging");
        }
        if (this.enemyLevel > 50){
            effects.push("Explosive", "Berserking", "Slowing", "Ethereal");
        }
        var effectsCount;
        if (this.enemyLevel < 25) effectsCount = Math.ceil((this.enemyLevel + 1) / 5);
        else effectsCount = 4 + (Math.ceil((this.enemyLevel - 19) / 10));
        var selectedEffects = [];
        var selectedEffectsCount = [];
        var maxEffectStack = 1;
        maxEffectStack += Math.floor(this.enemyLevel / 10);
        var healthMult = 1;
        var attackMult = 1;
        for (var x = 0; x < effectsCount; x++){
            var roll = getRandomIntSeeded(seed++, 0, effects.length);
            var effect = effects[roll];
            if (x == 0 && this.enemyLevel > 150){
                var immunities = ["Poison Immune", "Shock Immune", "Bleed Immune"];
                effect = immunities[(this.enemyLevel - 151) % 3];
            }
            if (!doubleResist && effect.search("Resistant") != -1){
                var offset = this.enemyLevel % 3;
                roll = getRandomIntSeeded(seed++, 0, 100);
                if (roll >= 40){
                    if (offset == 0) effect = "Poison Resistant";
                    if (offset == 1) effect = "Shock Resistant";
                    if (offset == 2) effect = "Bleed Resistant";
                }
            }
            var checkSelected = selectedEffects.indexOf(effect);
            if (checkSelected == -1){
                selectedEffects.push(effect);
                selectedEffectsCount.push(1);
                checkSelected = selectedEffects.length - 1;
            }
            else {
                selectedEffectsCount[checkSelected]++;
            }
            if (selectedEffectsCount[checkSelected] >= maxEffectStack) {
                effects.splice(effects.indexOf(effect), 1);
            }
            var totalStacks = selectedEffectsCount[checkSelected];
            var repeatMod = 1;
            if (totalStacks > 1){
                repeatMod *= Math.pow(0.5, (totalStacks - 1));
            }
            switch(effect){
                case "Healthy":
                    var mod = this.enemyLevel / 30;
                    healthMult += Math.min(1, mod);
                    if (selectedEffectsCount[checkSelected] >= 4) effects.splice(effects.indexOf(effect), 1);
                    break;
                case "Strong":
                    var mod = this.enemyLevel / 30;
                    attackMult += Math.min(1, mod);
                    if (selectedEffectsCount[checkSelected] >= 4) effects.splice(effects.indexOf(effect), 1);
                    break;
                case "Fast":
                    var mod = Math.pow(0.98, this.enemyLevel);
                    this.enemy.baseAttackSpeed *= Math.max(0.5, mod);
                    if (selectedEffectsCount[checkSelected] >= 2) effects.splice(effects.indexOf(effect), 1);
                    break;
                case "Poisoning":
                    this.enemy.poisonChance += Math.ceil(this.enemyLevel * 3 * repeatMod);
                    this.enemy.poisonMod += (Math.ceil((this.enemyLevel / 5) * repeatMod));
                    if (this.enemyLevel >= 30) this.enemy.poisonMod += (this.enemyLevel - 29);
                    if (totalStacks == 1) this.enemy.poisonStack += (Math.floor(this.enemyLevel / 10));
                    else this.enemy.poisonStack++;
                    this.enemy.poisonTime = 2500 + (Math.ceil(this.enemyLevel / 5) * 2500);
                    break;
                case "Bloodletting":
                    this.enemy.bleedChance += Math.ceil(this.enemyLevel * 3 * repeatMod);
                    this.enemy.bleedMod += Math.ceil(Math.min(2, (this.enemyLevel / 20)) * repeatMod);
                    this.enemy.bleedTime = 8000;
                    break;
                case "Shocking":
                    this.enemy.shockChance += Math.ceil(this.enemyLevel * 3 * repeatMod);
                    this.enemy.shockMod += Math.ceil(Math.min(2.5, (this.enemyLevel / 15)) * repeatMod);
                    this.enemy.shockTime = 8000;
                    break;
                case "Poison Resistant":
                    this.enemy.poisonResist += (10 * this.enemyLevel);
                    effects.splice(effects.indexOf(effect), 1);
                    if (!doubleResist || selectedEffects.indexOf('Bleed Resistant') != -1) effects.splice(effects.indexOf('Shock Resistant'), 1);
                    if (!doubleResist || selectedEffects.indexOf('Shock Resistant') != -1) effects.splice(effects.indexOf('Bleed Resistant'), 1);
                    break;
                case "Bleed Resistant":
                    this.enemy.bleedResist += (10 * this.enemyLevel);
                    effects.splice(effects.indexOf(effect), 1);
                    if (!doubleResist || selectedEffects.indexOf('Poison Resistant') != -1) effects.splice(effects.indexOf('Shock Resistant'), 1);
                    if (!doubleResist || selectedEffects.indexOf('Shock Resistant') != -1) effects.splice(effects.indexOf('Poison Resistant'), 1);
                    break;
                case "Shock Resistant":
                    this.enemy.shockResist += (10 * this.enemyLevel);
                    effects.splice(effects.indexOf(effect), 1);
                    if (!doubleResist || selectedEffects.indexOf('Bleed Resistant') != -1) effects.splice(effects.indexOf('Poison Resistant'), 1);
                    if (!doubleResist || selectedEffects.indexOf('Poison Resistant') != -1) effects.splice(effects.indexOf('Bleed Resistant'), 1);
                    break;
                case "Poison Immune":
                    this.enemy.immune = "poison";
                    effects.splice(effects.indexOf('Poison Resistant'), 1);
                    break;
                case "Shock Immune":
                    this.enemy.immune = "shock";
                    effects.splice(effects.indexOf('Shock Resistant'), 1);
                    break;
                case "Bleed Immune":
                    this.enemy.immune = "bleed";
                    effects.splice(effects.indexOf('Bleed Resistant'), 1);
                    break;
                case "Defensive":
                    this.enemy.defense += Math.ceil((this.enemy.level * 0.75) * Math.pow(1.05, this.enemy.level));
                    break;
                case "Lifestealing":
                    this.enemy.lifesteal += Math.min(1, (this.enemyLevel / 50));
                    break;
                case "Enraging":
                    this.enemy.enrageFreq -= 10;
                    this.enemy.enrageMult += 0.1;
                    if (selectedEffectsCount[checkSelected] >= 2) effects.splice(effects.indexOf(effect), 1);
                    break;
                case "Slowing":
                    this.enemy.slowAura += 0.1;

                    break;
                case "Explosive":
                    var count = selectedEffectsCount[checkSelected];
                    if (count >= 3) effects.splice(effects.indexOf(effect), 1);
                    if (count == 1) {
                        this.enemy.explodeDamage = 1.5;
                        this.enemy.explodeFreq = 20000;
                    }
                    else {
                        this.enemy.explodeDamage += 0.3;
                        this.enemy.explodeFreq -= 5000;
                    }
                    effects.splice(effects.indexOf('Berserking'));
                    effects.splice(effects.indexOf('Ethereal'));
                    break;
                case "Berserking":
                    var count = selectedEffectsCount[checkSelected];
                    if (count >= 3) effects.splice(effects.indexOf(effect), 1);
                    if (count == 1) {
                        this.enemy.berserkMod = 1.05;
                        this.enemy.berserkEvery = 4;
                    }
                    else {
                        this.enemy.berserkMod += 0.05;
                        this.enemy.berserkEvery--;
                    }
                    effects.splice(effects.indexOf('Explosive'));
                    effects.splice(effects.indexOf('Ethereal'));
                    break;
                case "Ethereal":
                    var count = selectedEffectsCount[checkSelected];
                    if (count >= 3) effects.splice(effects.indexOf(effect), 1);
                    if (count == 1) {
                        this.enemy.ethChance = 10;
                    }
                    else {
                        this.enemy.ethChance += 5;
                    }
                    effects.splice(effects.indexOf('Explosive'));
                    effects.splice(effects.indexOf('Berserking'));
                    break;
            }
        }
        this.enemy.baseHealth *= healthMult;
        this.enemy.baseAttack *= attackMult;
        for (var x = 0; x < selectedEffects.length; x++){
            this.profile += selectedEffects[x];
            if (selectedEffectsCount[x] > 1) this.profile += " x" + selectedEffectsCount[x] + "";
            this.profile += ", ";
        }
        this.profile = this.profile.substring(0, this.profile.length - 2)
    },
    trimpDied: function(){
        this.sessionTrimpsKilled++;
        this.lootAvg.counter += this.battleTime;
        this.resetCombat();
        if (this.sessionEnemiesKilled < this.sessionTrimpsKilled) swapClass('abTab', 'abTabLosing', document.getElementById('autoBattleTab'));
        this.popup(true, true);
        //this.notes += "Trimp Died. "
    },
    getDustMult: function(){
        var amt = 1;
        if (this.items.Lifegiving_Gem.equipped){
            amt *= (1 + this.items.Lifegiving_Gem.dustIncrease());
        }
        if (this.items.Gaseous_Greataxe.equipped){
            amt *= this.items.Gaseous_Greataxe.dustMult();
        }
        amt += this.trimp.dustMult;
        if (this.oneTimers.Dusty_Tome.owned){
            amt *= (1 + (0.05 * (this.maxEnemyLevel - 1)));
        }
        if (u2Mutations.tree.Dust.purchased){
            var mutMult = 1.25;
            if (u2Mutations.tree.Dust2.purchased){
                mutMult += 0.25;
            }
            amt *= mutMult;
        }
        if (this.items.Box_of_Spores.equipped && !this.enemy.hadBleed && this.enemy.poison.time > 0){
            
            amt *= this.items.Box_of_Spores.dustMult();
        }
        if (game.global.fluffyExp2 >= 1466015503701000) amt *= 5; //don't even look at this line, just move on
        return amt;
    },
    getEnrageMult: function(){
        var enrages = Math.floor(this.battleTime / (this.enemy.enrageFreq * 1000));
        if (enrages < 1) return 1;
        return Math.pow(this.enemy.enrageMult, enrages);
    },
    getDustReward: function(){
        var amt = (1 + ((this.enemy.level - 1) * 5)) * Math.pow(1.19, (this.enemy.level - 1));
        if (this.enemy.level >= 50) amt *= Math.pow(1.1, (this.enemy.level - 49));
        amt *= this.getDustMult();
        return amt;
    },
    enemyDied: function(){
        //this.notes += "Enemy Died. "
        this.sessionEnemiesKilled++;
        game.stats.saKills.value++;
        if (this.sessionEnemiesKilled >= 100 && this.sessionTrimpsKilled == 0 && this.enemyLevel >= 5) giveSingleAchieve("Huffstle");
        if (this.enemyLevel >= 20 && this.trimp.shockTime <= 0 && this.trimp.bleedTime <= 0 && this.trimp.poisonTime <= 0) giveSingleAchieve("Just Smack It");
        if (this.enemyLevel >= 100) giveSingleAchieve("Heroic Huffy");
        var amt = this.getDustReward();
        if (this.speed == 1) {
            this.dust += amt;
            game.stats.saDust.value += amt;
            if (this.enemyLevel > 50){
                this.shardDust += amt;
                if (this.shardDust >= 1e9){
                    var shardAmt = Math.floor(this.shardDust / 1e9);
                    this.shards += shardAmt;
                    this.shardDust -= 1e9 * shardAmt;
                    game.stats.saShards.value += shardAmt;
                }
            }
            
        }
        this.lootAvg.accumulator += amt;
        this.lootAvg.counter += this.battleTime;
        if (!this.lastFightDurations) this.lastFightDurations = [];
        this.lastFightDurations.push(this.battleTime / 1000);
        if (this.lastFightDurations.length > 100) this.lastFightDurations.shift();
        if (this.enemy.level == this.maxEnemyLevel && this.speed == 1 && this.maxEnemyLevel <= 160){
            this.enemiesKilled++;
            if (this.enemiesKilled >= this.nextLevelCount()) {
                if (this.maxEnemyLevel == 160){
                    this.bonuses.Radon.level = 52;
                    this.bonuses.Stats.level = 52;
                    this.bonuses.Scaffolding.level = 20;
                    this.canSeal = true;
                }
                else{
                    this.maxEnemyLevel++;
                    game.stats.saHighestLevel.valueTotal = this.maxEnemyLevel;
                    if (this.autoLevel) this.enemyLevel++;
                    this.enemiesKilled = 0;
                    this.resetStats();
                }
            }
        }
        if (this.sessionEnemiesKilled > this.sessionTrimpsKilled) swapClass('abTab', 'abTabWinning', document.getElementById('autoBattleTab'));
        this.resetCombat();
        this.checkLastActions();
        this.popup(true, false, false, false, true);
    },
    nextLevelCount: function(){
        if (this.enemyLevel < 20) return 10 * this.enemyLevel;
        return (190 + (15 * (this.enemyLevel - 19)))
    },
    update: function(){
        if (game.global.highestRadonLevelCleared < 74) return;
        if (this.sealed) return;
        if (usingRealTimeOffline && this.speed > 1){
            this.settings.practice.enabled = 0;
            this.speed = 1;
        }
        for (var x = 0; x < this.speed; x++){
            this.fight();
            this.popup(true, true);
            this.battleTime += this.frameTime;
        }
    },    
    getMaxItems: function(){
        return this.maxItems + this.bonuses.Extra_Limbs.level;
    },
    getDustPs: function() {
        var dps = 0;
        if (this.lootAvg.accumulator == 0){
            if (!this.enemy) return 0;
            if (this.sessionTrimpsKilled > 0) return 0;
            if (this.enemy.health >= this.enemy.maxHealth) return 0;
            if (this.enemy.health <= 0 || this.trimp.health <= 0) return 0;
            var enPct = this.enemy.health / this.enemy.maxHealth;
            var tpPct = this.trimp.health / this.trimp.maxHealth;
            if (enPct > tpPct) return 0;
            var reward = this.getDustReward();
            dps = (1000 * ((reward * (1 - enPct)) / this.battleTime));
        }
        else dps = (1000 * (this.lootAvg.accumulator / this.lootAvg.counter));
        if (dps < 0.01) dps = 0;
        return dps;
    },
    resetStats: function(){
        this.sessionEnemiesKilled = 0;
        this.sessionTrimpsKilled = 0;
        this.lootAvg.accumulator = 0;
        this.lootAvg.counter = 0;
        this.battleTime = 0;
        this.lastFightDurations = [];
        swapClass('abTab', 'abTabNone', document.getElementById('autoBattleTab'));
    },
    getFreshRings: function(){
        return {
            level: 1,
            mods: []
        };
    },
    getRingStatusDamage: function(){
        if (!this.oneTimers.The_Ring.owned) return 0;
        return this.rings.level * 25 * Math.pow(1.5, Math.floor(this.rings.level / 10));
    },
    getRingPoisonDamage: function(){
        if (!this.oneTimers.The_Ring.owned) return 0;
        return this.rings.level * 15 * Math.pow(5, Math.floor(this.rings.level / 10));
    },
    getRingStatusChance: function(){
        if (this.rings.level < 10) return 0;
        var calcLevel = this.rings.level - 9;
        return calcLevel * 20 * Math.pow(1.25, Math.floor(calcLevel / 10));
    },
    getRingStatAmt: function(modObj){
        return modObj.baseGain * this.rings.level * Math.pow(modObj.perTen, Math.floor(this.rings.level / 10));
    },
    getRingSlots: function(){
        var amt = Math.floor(this.rings.level / 15) + 1;
        if (amt > 3) amt = 3;
        return amt;
    },
    getAvailableRingMods: function(){
        var availableMods = [];
        var keys = Object.keys(autoBattle.ringStats);
        for (var x = 0; x < keys.length; x++){
            if (this.rings.mods.indexOf(keys[x]) == -1) availableMods.push(keys[x]);
        }
        return availableMods;
    },
    getRingLevelCost: function(){
        return Math.ceil(15 * Math.pow(2, this.rings.level));
    },
    countEquippedItems: function(){
        var count = 0;
        for (var ck in this.items){
            if (this.items[ck].equipped) count++;
        }
        return count;
    },
    countOwnedItems: function(){
        var count = 0;
        for (var ck in this.items){
            if (this.items[ck].owned) count++;
        }
        return count;
    },
    getBonusCost: function(what){
        var bonus = this.bonuses[what];
        return Math.ceil(bonus.price * Math.pow(bonus.priceMod, bonus.level));
    },
    upgradeCost: function(item){
        var itemObj = this.items[item];
        if (!itemObj) return;
        var priceMod = itemObj.priceMod || 3;
        var startPrice = itemObj.startPrice || 5;
        return startPrice * Math.pow(priceMod, itemObj.level - 1);
    },
    cleanName: function(name){
        return name.split("__").join("-").split("_").join(" ");
    },
    ringStats: {
            attack: {
                formatEffect: function(amt){
                    return "+ " + prettify(amt) + " Attack";
                },
                name: "Attack",
                doStuff: function(amt){
                    autoBattle.trimp.attack += amt;
                },
                baseGain: 50,
                perTen: 5
            },
            health: {
                formatEffect: function(amt){
                    return "+ " + prettify(amt) + " Health";
                },
                name: "Health",
                doStuff: function(amt){
                    autoBattle.trimp.maxHealth += amt;
                },
                baseGain: 200,
                perTen: 5,
            },
            defense: {
                formatEffect: function(amt){
                    return "+ " + prettify(amt) + " Defense";
                },
                name: "Defense",
                doStuff: function(amt){
                    autoBattle.trimp.defense += amt;
                },
                baseGain: 25,
                perTen: 2.5,
            },
            lifesteal: {
                formatEffect: function(amt){
                    return "+ " + prettify(amt) + "% Lifesteal";
                },
                name: "Lifesteal",
                doStuff: function(amt){
                    amt /= 100;
                    autoBattle.trimp.lifesteal += amt;
                },
                baseGain: 2.5,
                perTen: 4,
            },
            dustMult: {
                formatEffect: function(amt){
                    return "+ " + prettify(amt) + "% Dust Mult";
                },
                name: "Dust Mult",
                doStuff: function(amt){
                    amt /= 100;
                    autoBattle.trimp.dustMult += amt;
                },
                baseGain: 5,
                perTen: 2,
            }
    },
  };
  return autoBattle;
}

    var ab = createRawAutoBattle(gameMock, u2MutationsMock);
    ab.popup = function() {};
    ab.checkLastActions = function() {};
    ab._gameMock = gameMock;
    ab._u2MutationsMock = u2MutationsMock;
    autoBattle = ab;
    ab._activate = function() { autoBattle = ab; };
    return ab;
  }

  var _sharedHeadlessAb = null;
  function getSharedHeadlessInstance() {
    if (!_sharedHeadlessAb) _sharedHeadlessAb = createSimInstance();
    return _sharedHeadlessAb;
  }

  // High-level SASimEngine API
  var SASimEngine = {
    createInstance: createSimInstance,

    unlockedSlotsForLevel: function(level) {
      var amt = Math.floor((Number(level) || 1) / 15) + 1;
      return Math.min(3, Math.max(1, amt));
    },

    getEnemyDef: function(level) {
      var lvl = Number(level) || 1;
      var needed = lvl < 20 ? (10 * lvl) : (190 + (15 * (lvl - 19)));
      return { level: lvl, neededKills: needed };
    },

    // Convert planner state / build into simulation configuration
    normalizeConfig: function(source) {
      var cfg = {
        level: 1,
        enemyLevel: 1,
        maxEnemyLevel: 1,
        extraLimbs: 0,
        items: {},
        equipped: [],
        ring: { level: 1, mods: [] },
        rings: { level: 1, mods: [] },
        bonuses: { Radon: 0, Stats: 0, Scaffolding: 0, Extra_Limbs: 0 },
        oneTimers: {},
        fluffyExp2: 0,
        u2Mutations: { Dust: false, Dust2: false }
      };
      cfg.rings = cfg.ring;

      if (!source) return cfg;

      var saveObj = source.save || (source.global && source) || null;
      var g = (saveObj && saveObj.global) || source.global || {};
      var abData = source.ab || (g && g.autoBattleData) || (source.global && source.global.autoBattleData) || source;

      var targetBuild = source.build || (source.i ? source : null);
      var lvl = Number(
        source.level ||
        source.enemyLevel ||
        (targetBuild && (targetBuild.saLevel || targetBuild.level)) ||
        (abData && (abData.enemyLevel || abData.maxEnemyLevel)) ||
        1
      );
      cfg.level = lvl;
      cfg.enemyLevel = lvl;
      cfg.maxEnemyLevel = Number(source.maxEnemyLevel || (abData && abData.maxEnemyLevel) || lvl);
      if (cfg.maxEnemyLevel < cfg.enemyLevel) cfg.maxEnemyLevel = cfg.enemyLevel;

      if (abData && abData.bonuses) {
        cfg.bonuses.Radon = abData.bonuses.Radon ? (abData.bonuses.Radon.level || abData.bonuses.Radon) : 0;
        cfg.bonuses.Stats = abData.bonuses.Stats ? (abData.bonuses.Stats.level || abData.bonuses.Stats) : 0;
        cfg.bonuses.Scaffolding = abData.bonuses.Scaffolding ? (abData.bonuses.Scaffolding.level || abData.bonuses.Scaffolding) : 0;
        cfg.bonuses.Extra_Limbs = abData.bonuses.Extra_Limbs ? (abData.bonuses.Extra_Limbs.level || abData.bonuses.Extra_Limbs) : 0;
      }
      cfg.extraLimbs = cfg.bonuses.Extra_Limbs || 0;

      if (abData && abData.oneTimers) {
        for (var ot in abData.oneTimers) {
          cfg.oneTimers[ot] = !!(abData.oneTimers[ot] === true || (abData.oneTimers[ot] && abData.oneTimers[ot].owned));
        }
      }

      if (g.fluffyExp2) cfg.fluffyExp2 = g.fluffyExp2;
      if (g.u2MutationData) {
        cfg.u2Mutations.Dust = !!g.u2MutationData.Dust;
        cfg.u2Mutations.Dust2 = !!g.u2MutationData.Dust2;
      }

      var rSource = (targetBuild && targetBuild.r) || source.r || source.ring || source.rings || (abData && (abData.rings || abData.ring)) || {};
      cfg.ring.level = Number(rSource.lv || rSource.level || 1);
      var rawMods = rSource.m || rSource.mods || [];
      cfg.ring.mods = rawMods.map(function(m) {
        var s = String(m).toLowerCase();
        if (s === 'dust' || s === 'dustmult') return 'Dust';
        if (s === 'attack') return 'Attack';
        if (s === 'defense') return 'Defense';
        if (s === 'health') return 'Health';
        if (s === 'lifesteal') return 'Lifesteal';
        return m.charAt(0).toUpperCase() + m.slice(1);
      });

      if (abData && abData.items) {
        for (var ik in abData.items) {
          var it = abData.items[ik];
          if (typeof it === 'number') {
            cfg.items[ik] = { level: it, equipped: false };
          } else if (it && typeof it === 'object') {
            var itLv = it.level || 1;
            var isEq = !!it.equipped;
            cfg.items[ik] = { level: itLv, equipped: isEq };
            if (isEq) cfg.equipped.push(ik);
          }
        }
      }

      if (source.equipped && Array.isArray(source.equipped)) {
        cfg.equipped = source.equipped.slice();
        for (var k in cfg.items) cfg.items[k].equipped = (cfg.equipped.indexOf(k) !== -1);
        for (var ei = 0; ei < cfg.equipped.length; ei++) {
          var eqK = cfg.equipped[ei];
          if (!cfg.items[eqK]) {
            cfg.items[eqK] = { level: 1, equipped: true };
          } else if (typeof cfg.items[eqK] === 'object') {
            cfg.items[eqK].equipped = true;
          }
        }
      }

      if (targetBuild && targetBuild.i) {
        cfg.equipped = [];
        for (var bk in cfg.items) cfg.items[bk].equipped = false;
        for (var reqKey in targetBuild.i) {
          var reqLv = targetBuild.i[reqKey];
          var existingLv = (cfg.items[reqKey] && cfg.items[reqKey].level) || 0;
          var finalLv = Math.max(existingLv, reqLv);
          if (typeof SA_DATA !== 'undefined' && SA_DATA.items && SA_DATA.items[reqKey]) {
            var itemMeta = SA_DATA.items[reqKey];
            if (itemMeta.noUp || (itemMeta.max && itemMeta.max <= 1)) finalLv = 1;
          }
          cfg.items[reqKey] = { level: finalLv, equipped: true };
          cfg.equipped.push(reqKey);
        }
      }

      return cfg;
    },

    // Apply configuration to an autoBattle instance
    applyConfig: function(ab, cfg) {
      if (ab._activate) ab._activate();
      ab.enemyLevel = Number(cfg.level || cfg.enemyLevel || 1);
      ab.maxEnemyLevel = Number(cfg.maxEnemyLevel || ab.enemyLevel);
      ab.autoLevel = false;
      ab.nextLevelCount = function() { return Infinity; };

      ab._gameMock.global.highestRadonLevelCleared = 205;
      ab._gameMock.global.fluffyExp2 = cfg.fluffyExp2 || 0;
      ab._u2MutationsMock.tree.Dust.purchased = !!(cfg.u2Mutations && cfg.u2Mutations.Dust);
      ab._u2MutationsMock.tree.Dust2.purchased = !!(cfg.u2Mutations && cfg.u2Mutations.Dust2);

      if (cfg.bonuses) {
        for (var bKey in ab.bonuses) {
          ab.bonuses[bKey].level = cfg.bonuses[bKey] || 0;
        }
      }

      if (cfg.oneTimers) {
        for (var otKey in ab.oneTimers) {
          ab.oneTimers[otKey].owned = !!cfg.oneTimers[otKey];
        }
      }

      var rObj = cfg.ring || cfg.rings;
      if (rObj) {
        ab.rings.level = rObj.level || 1;
        var rMods = (rObj.mods || []).slice();
        ab.rings.mods = rMods.map(function(m) {
          var s = String(m).toLowerCase();
          if (s === 'dust' || s === 'dustmult') return 'dustMult';
          if (s === 'attack') return 'attack';
          if (s === 'defense') return 'defense';
          if (s === 'health') return 'health';
          if (s === 'lifesteal') return 'lifesteal';
          return m;
        });
        ab.ring = ab.rings;
      }

      var eqSet = {};
      if (Array.isArray(cfg.equipped)) {
        cfg.equipped.forEach(function(k) { eqSet[k] = true; });
      }

      for (var itKey in ab.items) {
        var it = ab.items[itKey];
        it.owned = true;
        var def = cfg.items && cfg.items[itKey];
        if (typeof def === 'number') {
          it.level = it.noUpgrade ? 1 : def;
          it.equipped = !!eqSet[itKey];
        } else if (def && typeof def === 'object') {
          it.level = it.noUpgrade ? 1 : (def.level || 1);
          it.equipped = (cfg.equipped && cfg.equipped.length) ? !!eqSet[itKey] : !!def.equipped;
        } else {
          it.level = 1;
          it.equipped = !!eqSet[itKey];
        }
      }

      var eqObjs = [];
      for (var eqKey in ab.items) {
        var eqItem = ab.items[eqKey];
        if (eqItem.equipped && eqItem.doStuff) eqObjs.push(eqItem);
      }
      ab._equippedItems = eqObjs;
      ab.resetCombat(true);
    },

    // Fast Headless Batch Runner
    runBatch: function(sourceConfig, options) {
      options = options || {};
      var maxKills = options.kills || options.maxKills || 100;
      var maxFights = options.maxFights || Math.max(maxKills * 2, 250);
      var maxTicks = options.ticks || Math.max(maxKills * 50, 100000);

      var ab = getSharedHeadlessInstance();
      var cfg = SASimEngine.normalizeConfig(sourceConfig);
      if (options.enemyLevel) { cfg.level = options.enemyLevel; cfg.enemyLevel = options.enemyLevel; }
      if (options.equipped) cfg.equipped = options.equipped;
      if (options.ringMods) cfg.rings.mods = options.ringMods;
      if (options.ringLevel) cfg.rings.level = options.ringLevel;

      SASimEngine.applyConfig(ab, cfg);
      ab.autoLevel = false;
      ab.nextLevelCount = function() { return Infinity; };

      var ticks = 0;
      var totalFights = 0;

      while (ab.sessionEnemiesKilled < maxKills && totalFights < maxFights && ticks < maxTicks) {
        ab.fight();
        ab.battleTime += ab.frameTime;
        ticks++;
        totalFights = ab.sessionEnemiesKilled + ab.sessionTrimpsKilled;
      }

      var dps = ab.getDustPs();
      var winRate = totalFights > 0 ? (ab.sessionEnemiesKilled / totalFights * 100) : 100;
      var totalSimSec = (ticks * ab.frameTime) / 1000;
      var secPerKill = ab.sessionEnemiesKilled > 0 ? (totalSimSec / ab.sessionEnemiesKilled) : 0;

      return {
        dustPs: dps,
        shardsPs: ab.enemyLevel > 50 ? (dps / 1e9) : 0,
        kills: ab.sessionEnemiesKilled,
        deaths: ab.sessionTrimpsKilled,
        totalFights: totalFights,
        winRate: winRate,
        winRateStr: winRate.toFixed(1) + '%',
        ticks: ticks,
        simSec: totalSimSec,
        secPerKill: secPerKill,
        timePerKill: secPerKill,
        enemyLevel: ab.enemyLevel
      };
    },

    // Monte Carlo Push Simulation across multiple seeds / fights
    monteCarloPush: function(sourceConfig, options) {
      options = options || {};
      var cfg = SASimEngine.normalizeConfig(sourceConfig);
      var targetLevel = options.targetLevel || cfg.level || cfg.enemyLevel || 1;
      var numFights = options.fights || 100;

      cfg.level = targetLevel;
      cfg.enemyLevel = targetLevel;
      if (options.equipped) cfg.equipped = options.equipped;
      if (options.items) {
        for (var k in options.items) {
          if (typeof options.items[k] === 'number') cfg.items[k] = { level: options.items[k], equipped: cfg.equipped.indexOf(k) !== -1 };
          else cfg.items[k] = options.items[k];
        }
      }
      if (options.rings || options.ring) {
        var r = options.rings || options.ring;
        cfg.ring.level = r.level || cfg.ring.level;
        if (r.mods) cfg.ring.mods = r.mods.slice();
      }

      var ab = getSharedHeadlessInstance();
      SASimEngine.applyConfig(ab, cfg);

      var fightDurations = [];
      var wins = 0;
      var losses = 0;
      var enrageCount = 0;
      var huffyHpPcts = [];
      var currentFightTicks = 0;
      var maxFightTicks = options.maxTicks || 3000;

      for (var f = 0; f < numFights; f++) {
        currentFightTicks = 0;
        var initialKills = ab.sessionEnemiesKilled;
        var initialDeaths = ab.sessionTrimpsKilled;

        while (ab.sessionEnemiesKilled === initialKills && ab.sessionTrimpsKilled === initialDeaths && currentFightTicks < maxFightTicks) {
          ab.fight();
          ab.battleTime += ab.frameTime;
          currentFightTicks++;
        }

        var durationSec = (currentFightTicks * ab.frameTime) / 1000;
        fightDurations.push(durationSec);
        if (ab.battleTime >= 60000) enrageCount++;

        var hpPct = (ab.trimp && ab.trimp.maxHealth > 0) ? Math.max(0, (ab.trimp.health / ab.trimp.maxHealth) * 100) : 0;
        huffyHpPcts.push(hpPct);

        if (ab.sessionEnemiesKilled > initialKills) wins++;
        else losses++;
      }

      var avgDuration = fightDurations.length ? (fightDurations.reduce(function(a, b) { return a + b; }, 0) / fightDurations.length) : 0;
      var winRate = numFights > 0 ? (wins / numFights * 100) : 0;
      var dps = ab.getDustPs();
      var secPerKill = wins > 0 ? (fightDurations.reduce(function(a, b){ return a + b; }, 0) / wins) : 0;
      var avgHp = huffyHpPcts.length ? (huffyHpPcts.reduce(function(a, b) { return a + b; }, 0) / huffyHpPcts.length) : 0;
      var minHp = huffyHpPcts.length ? Math.min.apply(null, huffyHpPcts) : 0;
      var maxHp = huffyHpPcts.length ? Math.max.apply(null, huffyHpPcts) : 0;

      return {
        targetLevel: targetLevel,
        fights: numFights,
        wins: wins,
        losses: losses,
        winRate: winRate,
        winRateStr: winRate.toFixed(1) + '%',
        avgDurationSec: avgDuration,
        avgFightDuration: avgDuration,
        dustPs: dps,
        secPerKill: secPerKill,
        timePerKill: secPerKill,
        enrageCount: enrageCount,
        avgHuffyHpPct: avgHp,
        minHuffyHpPct: minHp,
        maxHuffyHpPct: maxHp,
        durations: fightDurations
      };
    },

    // Ring Modifier Permutation Optimizer
    optimizeRing: function(sourceConfig, options) {
      options = options || {};
      var cfg = SASimEngine.normalizeConfig(sourceConfig);
      var slotCount = options.slots || SASimEngine.unlockedSlotsForLevel(cfg.ring.level);
      var enemyLevel = options.enemyLevel || cfg.level || cfg.enemyLevel || 1;

      var availableMods = ['Attack', 'Dust', 'Defense', 'Health', 'Lifesteal'];
      var perms = [];

      function getCombinations(arr, k, start, current) {
        if (current.length === k) {
          perms.push(current.slice());
          return;
        }
        for (var i = start; i < arr.length; i++) {
          current.push(arr[i]);
          getCombinations(arr, k, i + 1, current);
          current.pop();
        }
      }

      getCombinations(availableMods, slotCount, 0, []);

      var results = perms.map(function(modCombo) {
        var testCfg = JSON.parse(JSON.stringify(cfg));
        testCfg.ring.mods = modCombo.slice();
        testCfg.rings.mods = modCombo.slice();
        var res = SASimEngine.runBatch(testCfg, {
          enemyLevel: enemyLevel,
          kills: options.maxKills || 25,
          maxFights: 35
        });
        return {
          mods: modCombo,
          dustPs: res.dustPs,
          winRate: res.winRate,
          timePerKill: res.secPerKill,
          secPerKill: res.secPerKill
        };
      });

      results.sort(function(a, b) { return b.dustPs - a.dustPs; });
      return results;
    },

    // Upgrade Sensitivity / ROI Evaluator
    evaluateUpgradeROI: function(sourceConfig, options) {
      options = options || {};
      var cfg = SASimEngine.normalizeConfig(sourceConfig);
      var baseline = SASimEngine.runBatch(cfg, { kills: options.maxKills || 30, maxFights: 40 });

      var results = [];
      var keys = cfg.equipped && cfg.equipped.length ? cfg.equipped : Object.keys(cfg.items);

      // 1. Test ring upgrade
      var ringCfg = JSON.parse(JSON.stringify(cfg));
      ringCfg.ring.level++;
      ringCfg.rings.level = ringCfg.ring.level;
      var ringSim = SASimEngine.runBatch(ringCfg, { kills: options.maxKills || 30, maxFights: 40 });
      var ringDelta = ringSim.dustPs - baseline.dustPs;
      var ringDeltaK = Math.max(0, (baseline.secPerKill || 0) - (ringSim.secPerKill || 0));
      results.push({
        type: 'ring',
        key: 'ring',
        label: 'The Ring Lv ' + cfg.ring.level + ' → ' + (cfg.ring.level + 1),
        cost: Math.ceil(15 * Math.pow(2, cfg.ring.level)),
        shards: true,
        currentLevel: cfg.ring.level,
        nextLevel: cfg.ring.level + 1,
        baselineDustPs: baseline.dustPs,
        newDustPs: ringSim.dustPs,
        deltaDustPs: ringDelta,
        baselineSecPerKill: baseline.secPerKill,
        newSecPerKill: ringSim.secPerKill,
        deltaSecPerKill: ringDeltaK,
        paybackSec: ringDelta > 0 ? ((Math.ceil(15 * Math.pow(2, cfg.ring.level)) * 1e9) / ringDelta) : Infinity,
        pctGain: baseline.dustPs > 0 ? ((ringDelta / baseline.dustPs) * 100) : 0
      });

      // 2. Test equipped items
      for (var i = 0; i < keys.length; i++) {
        var k = keys[i];
        if (typeof SA_DATA !== 'undefined' && SA_DATA.items && SA_DATA.items[k]) {
          if (SA_DATA.items[k].noUp || (SA_DATA.items[k].max && SA_DATA.items[k].max <= 1)) continue;
        }
        var itemObj = cfg.items[k] || { level: 1, equipped: true };
        var curLv = itemObj.level || 1;
        var testCfg = JSON.parse(JSON.stringify(cfg));
        if (!testCfg.items[k] || typeof testCfg.items[k] !== 'object') {
          testCfg.items[k] = { level: curLv, equipped: true };
        }
        testCfg.items[k].level = curLv + 1;

        var sim = SASimEngine.runBatch(testCfg, { kills: options.maxKills || 30, maxFights: 40 });
        var delta = sim.dustPs - baseline.dustPs;
        var deltaK = Math.max(0, (baseline.secPerKill || 0) - (sim.secPerKill || 0));
        var label = (typeof itemName === 'function' ? itemName(k) : k.replace(/_/g, ' ')) + ' Lv ' + curLv + ' → ' + (curLv + 1);

        var itemCost = 0;
        var itemIsShard = false;
        if (typeof SA_DATA !== 'undefined' && SA_DATA.items && SA_DATA.items[k]) {
          var itemDef = SA_DATA.items[k];
          itemCost = (!itemDef.noUp) ? (itemDef.start * Math.pow(itemDef.mod, Math.max(0, curLv - 1))) : 0;
          itemIsShard = itemDef.dust === 'shards';
        } else {
          var abInst = getSharedHeadlessInstance();
          var sItem = abInst.items[k];
          if (sItem) {
            itemCost = (sItem.startPrice || 5) * Math.pow(sItem.priceMod || 3, Math.max(0, curLv - 1));
            itemIsShard = sItem.dustType === 'shards';
          }
        }

        var costInDust = itemIsShard ? (itemCost * 1e9) : itemCost;
        var itemPaybackSec = (delta > 0 && costInDust > 0) ? (costInDust / delta) : (delta > 0 ? 0 : Infinity);

        results.push({
          type: 'item',
          key: k,
          label: label,
          cost: itemCost,
          shards: itemIsShard,
          currentLevel: curLv,
          nextLevel: curLv + 1,
          baselineDustPs: baseline.dustPs,
          newDustPs: sim.dustPs,
          deltaDustPs: delta,
          baselineSecPerKill: baseline.secPerKill,
          newSecPerKill: sim.secPerKill,
          deltaSecPerKill: deltaK,
          paybackSec: itemPaybackSec,
          pctGain: baseline.dustPs > 0 ? ((delta / baseline.dustPs) * 100) : 0
        });
      }

      // Sort by ROI payback (shortest payback time first, with non-positive gains at end)
      results.sort(function(a, b) {
        var aFinite = a.paybackSec != null && isFinite(a.paybackSec) && a.paybackSec > 0;
        var bFinite = b.paybackSec != null && isFinite(b.paybackSec) && b.paybackSec > 0;
        if (aFinite && bFinite) {
          if (Math.abs(a.paybackSec - b.paybackSec) > 0.001) return a.paybackSec - b.paybackSec;
          return b.deltaDustPs - a.deltaDustPs;
        }
        if (aFinite) return -1;
        if (bFinite) return 1;
        return b.deltaDustPs - a.deltaDustPs;
      });
      return results;
    },

    // Push Viability Heatmap Scanner (10-run smoke test, scanning 10 levels below highest winning level)
    scanPushHeatmap: function(sourceConfig, options) {
      options = options || {};
      var fightsPerLevel = options.fights || 500;
      var cfg = SASimEngine.normalizeConfig(sourceConfig);

      // 1. Smoke test: 10 runs per level across all levels (1 to 150) to quickly find highest level with a win
      var highestWinLevel = 1;
      var consecutiveZero = 0;
      var probeFights = 10;
      for (var lvl = 1; lvl <= 150; lvl++) {
        var probeRes = SASimEngine.monteCarloPush(cfg, { targetLevel: lvl, fights: probeFights, maxTicks: 1500 });
        if (probeRes.wins > 0) {
          highestWinLevel = lvl;
          consecutiveZero = 0;
        } else {
          consecutiveZero++;
          if (lvl > (cfg.level || 1) + 15 && consecutiveZero >= 12) break;
        }
      }

      // 2. Begin 500-run scan 10 levels below the highest level with a win
      var endLvl = highestWinLevel;
      var startLvl = Math.max(1, endLvl - 9);
      var levels = [];
      var maxWin50 = startLvl;

      for (var l = startLvl; l <= endLvl; l++) {
        // Quick 30-fight check: if unwinnable (0 wins), skip remaining 470 ticks to keep UI lightning fast
        var quick = SASimEngine.monteCarloPush(cfg, { targetLevel: l, fights: 30, maxTicks: 1500 });
        var sim;
        if (quick.wins === 0) {
          sim = quick;
          sim.fights = fightsPerLevel;
          sim.wins = 0;
          sim.losses = fightsPerLevel;
          sim.winRate = 0;
        } else {
          sim = SASimEngine.monteCarloPush(cfg, { targetLevel: l, fights: fightsPerLevel, maxTicks: 1800 });
        }
        if (sim.winRate >= 50) {
          if (l > maxWin50) maxWin50 = l;
        }
        levels.push(sim);
      }

      return {
        highestWinLevel: highestWinLevel,
        maxWin50: maxWin50,
        rangeStart: startLvl,
        rangeEnd: endLvl,
        levels: levels
      };
    },

    // Stateful visual combat session
    createSession: function(sourceConfig) {
      var ab = createSimInstance();
      var cfg = SASimEngine.normalizeConfig(sourceConfig);
      var baseline = null;
      try {
        baseline = SASimEngine.runBatch(cfg, { maxKills: 15, maxFights: 25 });
      } catch (e) {
        baseline = null;
      }
      SASimEngine.applyConfig(ab, cfg);

      var eventLogs = [];
      function logEvent(text, type) {
        eventLogs.push({ time: ab.battleTime, text: text, type: type || 'info' });
        if (eventLogs.length > 50) eventLogs.shift();
      }

      var winningFightDurations = [];
      var completedFightDurations = [];

      return {
        ab: ab,
        cfg: cfg,
        logs: eventLogs,

        setConfig: function(newCfg) {
          cfg = SASimEngine.normalizeConfig(newCfg);
          eventLogs.length = 0;
          winningFightDurations = [];
          completedFightDurations = [];
          try {
            baseline = SASimEngine.runBatch(cfg, { maxKills: 15, maxFights: 25 });
          } catch (e) {
            baseline = null;
          }
          SASimEngine.applyConfig(ab, cfg);
        },

        reset: function(resetStats) {
          if (ab._activate) ab._activate();
          ab.resetCombat(resetStats);
          if (resetStats) {
            eventLogs.length = 0;
            winningFightDurations = [];
            completedFightDurations = [];
          }
        },

        step: function(ticks) {
          if (ab._activate) ab._activate();
          ticks = ticks || 1;
          for (var i = 0; i < ticks; i++) {
            var prevKills = ab.sessionEnemiesKilled;
            var prevDeaths = ab.sessionTrimpsKilled;
            var battleTimeBefore = ab.battleTime;

            ab.fight();
            ab.battleTime += ab.frameTime;

            if (ab.sessionEnemiesKilled > prevKills) {
              var killDuration = (ab.lastFightDurations && ab.lastFightDurations.length > 0)
                ? ab.lastFightDurations[ab.lastFightDurations.length - 1]
                : (battleTimeBefore / 1000);
              winningFightDurations.push(killDuration);
              if (winningFightDurations.length > 100) winningFightDurations.shift();
              completedFightDurations.push(killDuration);
              if (completedFightDurations.length > 100) completedFightDurations.shift();
            } else if (ab.sessionTrimpsKilled > prevDeaths) {
              var deathDuration = battleTimeBefore / 1000;
              completedFightDurations.push(deathDuration);
              if (completedFightDurations.length > 100) completedFightDurations.shift();
            }
          }
        },

        getState: function() {
          if (ab._activate) ab._activate();
          var trimp = ab.trimp || {};
          var enemy = ab.enemy || {};
          var totalFights = ab.sessionEnemiesKilled + ab.sessionTrimpsKilled;
          var winRate = totalFights > 0
            ? (ab.sessionEnemiesKilled / totalFights * 100)
            : (baseline ? baseline.winRate : 100);
          var dps = ab.getDustPs();
          if ((!dps || dps <= 0) && totalFights === 0 && baseline && baseline.dustPs) {
            dps = baseline.dustPs;
          }

          var currentFightSec = (ab.battleTime || 0) / 1000;

          var avgKillSec = null;
          if (winningFightDurations.length > 0) {
            var durSum = 0;
            for (var di = 0; di < winningFightDurations.length; di++) durSum += winningFightDurations[di];
            avgKillSec = durSum / winningFightDurations.length;
          } else if (ab.lastFightDurations && ab.lastFightDurations.length > 0) {
            var durSum2 = 0;
            for (var di2 = 0; di2 < ab.lastFightDurations.length; di2++) durSum2 += ab.lastFightDurations[di2];
            avgKillSec = durSum2 / ab.lastFightDurations.length;
          } else if (baseline && baseline.kills > 0 && baseline.secPerKill > 0) {
            avgKillSec = baseline.secPerKill;
          }

          var avgAllSec = null;
          if (completedFightDurations.length > 0) {
            var durSumAll = 0;
            for (var dai = 0; dai < completedFightDurations.length; dai++) durSumAll += completedFightDurations[dai];
            avgAllSec = durSumAll / completedFightDurations.length;
          } else if (baseline && baseline.simSec && baseline.totalFights) {
            avgAllSec = baseline.simSec / baseline.totalFights;
          }

          return {
            battleTimeMs: ab.battleTime,
            frameTimeMs: ab.frameTime,
            currentFightSec: currentFightSec,
            time: currentFightSec,
            enemyLevel: ab.enemyLevel,
            maxEnemyLevel: ab.maxEnemyLevel,
            dustPs: dps,
            shardsPs: ab.enemyLevel > 50 ? (dps / 1e9) : 0,
            sessionKills: ab.sessionEnemiesKilled,
            sessionDeaths: ab.sessionTrimpsKilled,
            wins: ab.sessionEnemiesKilled,
            losses: ab.sessionTrimpsKilled,
            totalFights: totalFights,
            winRate: winRate,
            secPerKill: avgKillSec,
            timePerKill: avgKillSec,
            avgLast100Sec: avgKillSec,
            avgFightDuration: avgAllSec,
            huffyHp: trimp.health || 0,
            huffyMaxHp: trimp.maxHealth || 1,
            huffyAtk: trimp.attack || 0,
            huffyDef: trimp.defense || 0,
            huffyLs: Math.round((trimp.lifesteal || 0) * 100),
            huffyDustMult: Math.round((ab.getDustMult() || 1) * 100),
            huffyAtkTimer: Math.max(0, (trimp.attackSpeed || 500) - (trimp.lastAttack || 0)),
            huffyMaxAtkTimer: trimp.attackSpeed || 500,
            huffyBuffs: {
              bleed: trimp.bleed && trimp.bleed.time > 0 ? { dmg: trimp.bleed.mod || 0, amp: Math.round((trimp.bleed.mod || 0) * 100) } : null,
              poison: trimp.poison && trimp.poison.time > 0 ? { stacks: trimp.poison.stacks || 0, amp: Math.round((trimp.poison.stacks || 0) * 100) } : null,
              shock: trimp.shock && trimp.shock.time > 0 ? { amp: Math.round((trimp.shock.mod || 0) * 100) } : null
            },
            enemyName: enemy.profile || ('Enemy Lv ' + ab.enemyLevel),
            enemyHp: enemy.health || 0,
            enemyMaxHp: enemy.maxHealth || 1,
            enemyAtk: enemy.attack || 0,
            enemyDef: enemy.defense || 0,
            enemyLs: Math.round((enemy.lifesteal || 0) * 100),
            enemyEnrageTimer: Math.max(0, 60000 - ab.battleTime),
            enemyAtkTimer: Math.max(0, (enemy.attackSpeed || 700) - (enemy.lastAttack || 0)),
            enemyMaxAtkTimer: enemy.attackSpeed || 700,
            enemyBuffs: {
              bleed: enemy.bleed && enemy.bleed.time > 0 ? { dmg: enemy.bleed.mod || 0, amp: Math.round((enemy.bleed.mod || 0) * 100) } : null,
              poison: enemy.poison && enemy.poison.time > 0 ? { stacks: enemy.poison.stacks || 0, amp: Math.round((enemy.poison.stacks || 0) * 100) } : null,
              shock: enemy.shock && enemy.shock.time > 0 ? { amp: Math.round((enemy.shock.mod || 0) * 100) } : null
            },
            huffy: {
              health: trimp.health || 0,
              maxHealth: trimp.maxHealth || 1,
              healthPct: trimp.maxHealth > 0 ? Math.max(0, Math.min(100, (trimp.health / trimp.maxHealth) * 100)) : 0,
              attack: trimp.attack || 0,
              attackSpeedMs: trimp.attackSpeed || 1,
              lastAttackMs: trimp.lastAttack || 0,
              attackPct: trimp.attackSpeed > 0 ? Math.min(100, (trimp.lastAttack / trimp.attackSpeed) * 100) : 0,
              defense: trimp.defense || 0,
              lifestealPct: (trimp.lifesteal || 0) * 100,
              bleedTime: (trimp.bleed && trimp.bleed.time) || 0,
              bleedMod: (trimp.bleed && trimp.bleed.mod) || 0,
              poisonTime: (trimp.poison && trimp.poison.time) || 0,
              poisonStacks: (trimp.poison && trimp.poison.stacks) || 0,
              shockTime: (trimp.shock && trimp.shock.time) || 0,
              shockMod: (trimp.shock && trimp.shock.mod) || 0,
              dustMultPct: (trimp.dustMult || 0) * 100
            },
            enemy: {
              name: enemy.profile || ('Enemy Lv ' + ab.enemyLevel),
              health: enemy.health || 0,
              maxHealth: enemy.maxHealth || 1,
              healthPct: enemy.maxHealth > 0 ? Math.max(0, Math.min(100, (enemy.health / enemy.maxHealth) * 100)) : 0,
              attack: enemy.attack || 0,
              attackSpeedMs: enemy.attackSpeed || 1,
              lastAttackMs: enemy.lastAttack || 0,
              attackPct: enemy.attackSpeed > 0 ? Math.min(100, (enemy.lastAttack / enemy.attackSpeed) * 100) : 0,
              defense: enemy.defense || 0,
              lifestealPct: (enemy.lifesteal || 0) * 100,
              bleedTime: (enemy.bleed && enemy.bleed.time) || 0,
              bleedMod: (enemy.bleed && enemy.bleed.mod) || 0,
              poisonTime: (enemy.poison && enemy.poison.time) || 0,
              poisonStacks: (enemy.poison && enemy.poison.stacks) || 0,
              shockTime: (enemy.shock && enemy.shock.time) || 0,
              shockMod: (enemy.shock && enemy.shock.mod) || 0,
              enrageMult: ab.getEnrageMult()
            }
          };
        }
      };
    }
  };


    root.SASimEngine = SASimEngine;
})(typeof window !== 'undefined' ? window : this);
