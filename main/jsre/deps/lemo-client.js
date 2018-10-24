(function (global, factory) {
    typeof exports === 'object' && typeof module !== 'undefined' ? module.exports = factory(require('bignumber.js')) :
        typeof define === 'function' && define.amd ? define(['bignumber.js'], factory) :
            (global.LemoClient = factory(global.BigNumber));
}(this, (function (BigNumber) { 'use strict';

    BigNumber = BigNumber && BigNumber.hasOwnProperty('default') ? BigNumber['default'] : BigNumber;

    function createCommonjsModule(fn, module) {
        return module = { exports: {} }, fn(module, module.exports), module.exports;
    }

    var _global = createCommonjsModule(function (module) {
        // https://github.com/zloirock/core-js/issues/86#issuecomment-115759028
        var global = module.exports = typeof window != 'undefined' && window.Math == Math
            ? window : typeof self != 'undefined' && self.Math == Math ? self
                // eslint-disable-next-line no-new-func
                : Function('return this')();
        if (typeof __g == 'number') __g = global; // eslint-disable-line no-undef
    });

    var _core = createCommonjsModule(function (module) {
        var core = module.exports = { version: '2.5.7' };
        if (typeof __e == 'number') __e = core; // eslint-disable-line no-undef
    });
    var _core_1 = _core.version;

    var _isObject = function (it) {
        return typeof it === 'object' ? it !== null : typeof it === 'function';
    };

    var _anObject = function (it) {
        if (!_isObject(it)) throw TypeError(it + ' is not an object!');
        return it;
    };

    var _fails = function (exec) {
        try {
            return !!exec();
        } catch (e) {
            return true;
        }
    };

    // Thank's IE8 for his funny defineProperty
    var _descriptors = !_fails(function () {
        return Object.defineProperty({}, 'a', { get: function () { return 7; } }).a != 7;
    });

    var document$1 = _global.document;
    // typeof document.createElement is 'object' in old IE
    var is = _isObject(document$1) && _isObject(document$1.createElement);
    var _domCreate = function (it) {
        return is ? document$1.createElement(it) : {};
    };

    var _ie8DomDefine = !_descriptors && !_fails(function () {
        return Object.defineProperty(_domCreate('div'), 'a', { get: function () { return 7; } }).a != 7;
    });

    // 7.1.1 ToPrimitive(input [, PreferredType])

    // instead of the ES6 spec version, we didn't implement @@toPrimitive case
    // and the second argument - flag - preferred type is a string
    var _toPrimitive = function (it, S) {
        if (!_isObject(it)) return it;
        var fn, val;
        if (S && typeof (fn = it.toString) == 'function' && !_isObject(val = fn.call(it))) return val;
        if (typeof (fn = it.valueOf) == 'function' && !_isObject(val = fn.call(it))) return val;
        if (!S && typeof (fn = it.toString) == 'function' && !_isObject(val = fn.call(it))) return val;
        throw TypeError("Can't convert object to primitive value");
    };

    var dP = Object.defineProperty;

    var f = _descriptors ? Object.defineProperty : function defineProperty(O, P, Attributes) {
        _anObject(O);
        P = _toPrimitive(P, true);
        _anObject(Attributes);
        if (_ie8DomDefine) try {
            return dP(O, P, Attributes);
        } catch (e) { /* empty */ }
        if ('get' in Attributes || 'set' in Attributes) throw TypeError('Accessors not supported!');
        if ('value' in Attributes) O[P] = Attributes.value;
        return O;
    };

    var _objectDp = {
        f: f
    };

    var _propertyDesc = function (bitmap, value) {
        return {
            enumerable: !(bitmap & 1),
            configurable: !(bitmap & 2),
            writable: !(bitmap & 4),
            value: value
        };
    };

    var _propertyDesc$1 = /*#__PURE__*/Object.freeze({
        default: _propertyDesc,
        __moduleExports: _propertyDesc
    });

    var descriptor = ( _propertyDesc$1 && _propertyDesc ) || _propertyDesc$1;

    var _hide = _descriptors ? function (object, key, value) {
        return _objectDp.f(object, key, descriptor(1, value));
    } : function (object, key, value) {
        object[key] = value;
        return object;
    };

    var hasOwnProperty = {}.hasOwnProperty;
    var _has = function (it, key) {
        return hasOwnProperty.call(it, key);
    };

    var id = 0;
    var px = Math.random();
    var _uid = function (key) {
        return 'Symbol('.concat(key === undefined ? '' : key, ')_', (++id + px).toString(36));
    };

    var _uid$1 = /*#__PURE__*/Object.freeze({
        default: _uid,
        __moduleExports: _uid
    });

    var uid = ( _uid$1 && _uid ) || _uid$1;

    var _redefine = createCommonjsModule(function (module) {
        var SRC = uid('src');
        var TO_STRING = 'toString';
        var $toString = Function[TO_STRING];
        var TPL = ('' + $toString).split(TO_STRING);

        _core.inspectSource = function (it) {
            return $toString.call(it);
        };

        (module.exports = function (O, key, val, safe) {
            var isFunction = typeof val == 'function';
            if (isFunction) _has(val, 'name') || _hide(val, 'name', key);
            if (O[key] === val) return;
            if (isFunction) _has(val, SRC) || _hide(val, SRC, O[key] ? '' + O[key] : TPL.join(String(key)));
            if (O === _global) {
                O[key] = val;
            } else if (!safe) {
                delete O[key];
                _hide(O, key, val);
            } else if (O[key]) {
                O[key] = val;
            } else {
                _hide(O, key, val);
            }
            // add fake Function#toString for correct work wrapped methods / constructors with methods like LoDash isNative
        })(Function.prototype, TO_STRING, function toString() {
            return typeof this == 'function' && this[SRC] || $toString.call(this);
        });
    });

    var _aFunction = function (it) {
        if (typeof it != 'function') throw TypeError(it + ' is not a function!');
        return it;
    };

    // optional / simple context binding

    var _ctx = function (fn, that, length) {
        _aFunction(fn);
        if (that === undefined) return fn;
        switch (length) {
            case 1: return function (a) {
                return fn.call(that, a);
            };
            case 2: return function (a, b) {
                return fn.call(that, a, b);
            };
            case 3: return function (a, b, c) {
                return fn.call(that, a, b, c);
            };
        }
        return function (/* ...args */) {
            return fn.apply(that, arguments);
        };
    };

    var PROTOTYPE = 'prototype';

    var $export = function (type, name, source) {
        var IS_FORCED = type & $export.F;
        var IS_GLOBAL = type & $export.G;
        var IS_STATIC = type & $export.S;
        var IS_PROTO = type & $export.P;
        var IS_BIND = type & $export.B;
        var target = IS_GLOBAL ? _global : IS_STATIC ? _global[name] || (_global[name] = {}) : (_global[name] || {})[PROTOTYPE];
        var exports = IS_GLOBAL ? _core : _core[name] || (_core[name] = {});
        var expProto = exports[PROTOTYPE] || (exports[PROTOTYPE] = {});
        var key, own, out, exp;
        if (IS_GLOBAL) source = name;
        for (key in source) {
            // contains in native
            own = !IS_FORCED && target && target[key] !== undefined;
            // export native or passed
            out = (own ? target : source)[key];
            // bind timers to global for call from export context
            exp = IS_BIND && own ? _ctx(out, _global) : IS_PROTO && typeof out == 'function' ? _ctx(Function.call, out) : out;
            // extend global
            if (target) _redefine(target, key, out, type & $export.U);
            // export
            if (exports[key] != out) _hide(exports, key, exp);
            if (IS_PROTO && expProto[key] != out) expProto[key] = out;
        }
    };
    _global.core = _core;
    // type bitmap
    $export.F = 1;   // forced
    $export.G = 2;   // global
    $export.S = 4;   // static
    $export.P = 8;   // proto
    $export.B = 16;  // bind
    $export.W = 32;  // wrap
    $export.U = 64;  // safe
    $export.R = 128; // real proto method for `library`
    var _export = $export;

    // 7.1.4 ToInteger
    var ceil = Math.ceil;
    var floor = Math.floor;
    var _toInteger = function (it) {
        return isNaN(it = +it) ? 0 : (it > 0 ? floor : ceil)(it);
    };

    var _toInteger$1 = /*#__PURE__*/Object.freeze({
        default: _toInteger,
        __moduleExports: _toInteger
    });

    var toInteger = ( _toInteger$1 && _toInteger ) || _toInteger$1;

    // 7.1.15 ToLength

    var min = Math.min;
    var _toLength = function (it) {
        return it > 0 ? min(toInteger(it), 0x1fffffffffffff) : 0; // pow(2, 53) - 1 == 9007199254740991
    };

    var toString = {}.toString;

    var _cof = function (it) {
        return toString.call(it).slice(8, -1);
    };

    var _library = false;

    var _shared = createCommonjsModule(function (module) {
        var SHARED = '__core-js_shared__';
        var store = _global[SHARED] || (_global[SHARED] = {});

        (module.exports = function (key, value) {
            return store[key] || (store[key] = value !== undefined ? value : {});
        })('versions', []).push({
            version: _core.version,
            mode: 'global',
            copyright: '© 2018 Denis Pushkarev (zloirock.ru)'
        });
    });

    var _wks = createCommonjsModule(function (module) {
        var store = _shared('wks');

        var Symbol = _global.Symbol;
        var USE_SYMBOL = typeof Symbol == 'function';

        var $exports = module.exports = function (name) {
            return store[name] || (store[name] =
                USE_SYMBOL && Symbol[name] || (USE_SYMBOL ? Symbol : uid)('Symbol.' + name));
        };

        $exports.store = store;
    });

    // 7.2.8 IsRegExp(argument)


    var MATCH = _wks('match');
    var _isRegexp = function (it) {
        var isRegExp;
        return _isObject(it) && ((isRegExp = it[MATCH]) !== undefined ? !!isRegExp : _cof(it) == 'RegExp');
    };

    // 7.2.1 RequireObjectCoercible(argument)
    var _defined = function (it) {
        if (it == undefined) throw TypeError("Can't call method on  " + it);
        return it;
    };

    // helper for String#{startsWith, endsWith, includes}



    var _stringContext = function (that, searchString, NAME) {
        if (_isRegexp(searchString)) throw TypeError('String#' + NAME + " doesn't accept regex!");
        return String(_defined(that));
    };

    var MATCH$1 = _wks('match');
    var _failsIsRegexp = function (KEY) {
        var re = /./;
        try {
            '/./'[KEY](re);
        } catch (e) {
            try {
                re[MATCH$1] = false;
                return !'/./'[KEY](re);
            } catch (f) { /* empty */ }
        } return true;
    };

    var STARTS_WITH = 'startsWith';
    var $startsWith = ''[STARTS_WITH];

    _export(_export.P + _export.F * _failsIsRegexp(STARTS_WITH), 'String', {
        startsWith: function startsWith(searchString /* , position = 0 */) {
            var that = _stringContext(this, searchString, STARTS_WITH);
            var index = _toLength(Math.min(arguments.length > 1 ? arguments[1] : undefined, that.length));
            var search = String(searchString);
            return $startsWith
                ? $startsWith.call(that, search, index)
                : that.slice(index, index + search.length) === search;
        }
    });

    // 22.1.3.31 Array.prototype[@@unscopables]
    var UNSCOPABLES = _wks('unscopables');
    var ArrayProto = Array.prototype;
    if (ArrayProto[UNSCOPABLES] == undefined) _hide(ArrayProto, UNSCOPABLES, {});
    var _addToUnscopables = function (key) {
        ArrayProto[UNSCOPABLES][key] = true;
    };

    var _addToUnscopables$1 = /*#__PURE__*/Object.freeze({
        default: _addToUnscopables,
        __moduleExports: _addToUnscopables
    });

    var _iterStep = function (done, value) {
        return { value: value, done: !!done };
    };

    var _iterators = {};

    // fallback for non-array-like ES3 and non-enumerable old V8 strings

    // eslint-disable-next-line no-prototype-builtins
    var _iobject = Object('z').propertyIsEnumerable(0) ? Object : function (it) {
        return _cof(it) == 'String' ? it.split('') : Object(it);
    };

    // to indexed object, toObject with fallback for non-array-like ES3 strings


    var _toIobject = function (it) {
        return _iobject(_defined(it));
    };

    var max = Math.max;
    var min$1 = Math.min;
    var _toAbsoluteIndex = function (index, length) {
        index = toInteger(index);
        return index < 0 ? max(index + length, 0) : min$1(index, length);
    };

    // false -> Array#indexOf
    // true  -> Array#includes



    var _arrayIncludes = function (IS_INCLUDES) {
        return function ($this, el, fromIndex) {
            var O = _toIobject($this);
            var length = _toLength(O.length);
            var index = _toAbsoluteIndex(fromIndex, length);
            var value;
            // Array#includes uses SameValueZero equality algorithm
            // eslint-disable-next-line no-self-compare
            if (IS_INCLUDES && el != el) while (length > index) {
                value = O[index++];
                // eslint-disable-next-line no-self-compare
                if (value != value) return true;
                // Array#indexOf ignores holes, Array#includes - not
            } else for (;length > index; index++) if (IS_INCLUDES || index in O) {
                if (O[index] === el) return IS_INCLUDES || index || 0;
            } return !IS_INCLUDES && -1;
        };
    };

    var shared = _shared('keys');

    var _sharedKey = function (key) {
        return shared[key] || (shared[key] = uid(key));
    };

    var arrayIndexOf = _arrayIncludes(false);
    var IE_PROTO = _sharedKey('IE_PROTO');

    var _objectKeysInternal = function (object, names) {
        var O = _toIobject(object);
        var i = 0;
        var result = [];
        var key;
        for (key in O) if (key != IE_PROTO) _has(O, key) && result.push(key);
        // Don't enum bug & hidden keys
        while (names.length > i) if (_has(O, key = names[i++])) {
            ~arrayIndexOf(result, key) || result.push(key);
        }
        return result;
    };

    var _objectKeysInternal$1 = /*#__PURE__*/Object.freeze({
        default: _objectKeysInternal,
        __moduleExports: _objectKeysInternal
    });

    // IE 8- don't enum bug keys
    var _enumBugKeys = (
        'constructor,hasOwnProperty,isPrototypeOf,propertyIsEnumerable,toLocaleString,toString,valueOf'
    ).split(',');

    var _enumBugKeys$1 = /*#__PURE__*/Object.freeze({
        default: _enumBugKeys,
        __moduleExports: _enumBugKeys
    });

    var $keys = ( _objectKeysInternal$1 && _objectKeysInternal ) || _objectKeysInternal$1;

    var enumBugKeys = ( _enumBugKeys$1 && _enumBugKeys ) || _enumBugKeys$1;

    // 19.1.2.14 / 15.2.3.14 Object.keys(O)



    var _objectKeys = Object.keys || function keys(O) {
        return $keys(O, enumBugKeys);
    };

    var _objectDps = _descriptors ? Object.defineProperties : function defineProperties(O, Properties) {
        _anObject(O);
        var keys = _objectKeys(Properties);
        var length = keys.length;
        var i = 0;
        var P;
        while (length > i) _objectDp.f(O, P = keys[i++], Properties[P]);
        return O;
    };

    var document$2 = _global.document;
    var _html = document$2 && document$2.documentElement;

    // 19.1.2.2 / 15.2.3.5 Object.create(O [, Properties])



    var IE_PROTO$1 = _sharedKey('IE_PROTO');
    var Empty = function () { /* empty */ };
    var PROTOTYPE$1 = 'prototype';

    // Create object with fake `null` prototype: use iframe Object with cleared prototype
    var createDict = function () {
        // Thrash, waste and sodomy: IE GC bug
        var iframe = _domCreate('iframe');
        var i = enumBugKeys.length;
        var lt = '<';
        var gt = '>';
        var iframeDocument;
        iframe.style.display = 'none';
        _html.appendChild(iframe);
        iframe.src = 'javascript:'; // eslint-disable-line no-script-url
        // createDict = iframe.contentWindow.Object;
        // html.removeChild(iframe);
        iframeDocument = iframe.contentWindow.document;
        iframeDocument.open();
        iframeDocument.write(lt + 'script' + gt + 'document.F=Object' + lt + '/script' + gt);
        iframeDocument.close();
        createDict = iframeDocument.F;
        while (i--) delete createDict[PROTOTYPE$1][enumBugKeys[i]];
        return createDict();
    };

    var _objectCreate = Object.create || function create(O, Properties) {
        var result;
        if (O !== null) {
            Empty[PROTOTYPE$1] = _anObject(O);
            result = new Empty();
            Empty[PROTOTYPE$1] = null;
            // add "__proto__" for Object.getPrototypeOf polyfill
            result[IE_PROTO$1] = O;
        } else result = createDict();
        return Properties === undefined ? result : _objectDps(result, Properties);
    };

    var def = _objectDp.f;

    var TAG = _wks('toStringTag');

    var _setToStringTag = function (it, tag, stat) {
        if (it && !_has(it = stat ? it : it.prototype, TAG)) def(it, TAG, { configurable: true, value: tag });
    };

    var IteratorPrototype = {};

    // 25.1.2.1.1 %IteratorPrototype%[@@iterator]()
    _hide(IteratorPrototype, _wks('iterator'), function () { return this; });

    var _iterCreate = function (Constructor, NAME, next) {
        Constructor.prototype = _objectCreate(IteratorPrototype, { next: descriptor(1, next) });
        _setToStringTag(Constructor, NAME + ' Iterator');
    };

    // 7.1.13 ToObject(argument)

    var _toObject = function (it) {
        return Object(_defined(it));
    };

    // 19.1.2.9 / 15.2.3.2 Object.getPrototypeOf(O)


    var IE_PROTO$2 = _sharedKey('IE_PROTO');
    var ObjectProto = Object.prototype;

    var _objectGpo = Object.getPrototypeOf || function (O) {
        O = _toObject(O);
        if (_has(O, IE_PROTO$2)) return O[IE_PROTO$2];
        if (typeof O.constructor == 'function' && O instanceof O.constructor) {
            return O.constructor.prototype;
        } return O instanceof Object ? ObjectProto : null;
    };

    var ITERATOR = _wks('iterator');
    var BUGGY = !([].keys && 'next' in [].keys()); // Safari has buggy iterators w/o `next`
    var FF_ITERATOR = '@@iterator';
    var KEYS = 'keys';
    var VALUES = 'values';

    var returnThis = function () { return this; };

    var _iterDefine = function (Base, NAME, Constructor, next, DEFAULT, IS_SET, FORCED) {
        _iterCreate(Constructor, NAME, next);
        var getMethod = function (kind) {
            if (!BUGGY && kind in proto) return proto[kind];
            switch (kind) {
                case KEYS: return function keys() { return new Constructor(this, kind); };
                case VALUES: return function values() { return new Constructor(this, kind); };
            } return function entries() { return new Constructor(this, kind); };
        };
        var TAG = NAME + ' Iterator';
        var DEF_VALUES = DEFAULT == VALUES;
        var VALUES_BUG = false;
        var proto = Base.prototype;
        var $native = proto[ITERATOR] || proto[FF_ITERATOR] || DEFAULT && proto[DEFAULT];
        var $default = $native || getMethod(DEFAULT);
        var $entries = DEFAULT ? !DEF_VALUES ? $default : getMethod('entries') : undefined;
        var $anyNative = NAME == 'Array' ? proto.entries || $native : $native;
        var methods, key, IteratorPrototype;
        // Fix native
        if ($anyNative) {
            IteratorPrototype = _objectGpo($anyNative.call(new Base()));
            if (IteratorPrototype !== Object.prototype && IteratorPrototype.next) {
                // Set @@toStringTag to native iterators
                _setToStringTag(IteratorPrototype, TAG, true);
                // fix for some old engines
                if (typeof IteratorPrototype[ITERATOR] != 'function') _hide(IteratorPrototype, ITERATOR, returnThis);
            }
        }
        // fix Array#{values, @@iterator}.name in V8 / FF
        if (DEF_VALUES && $native && $native.name !== VALUES) {
            VALUES_BUG = true;
            $default = function values() { return $native.call(this); };
        }
        // Define iterator
        if (BUGGY || VALUES_BUG || !proto[ITERATOR]) {
            _hide(proto, ITERATOR, $default);
        }
        // Plug for library
        _iterators[NAME] = $default;
        _iterators[TAG] = returnThis;
        if (DEFAULT) {
            methods = {
                values: DEF_VALUES ? $default : getMethod(VALUES),
                keys: IS_SET ? $default : getMethod(KEYS),
                entries: $entries
            };
            if (FORCED) for (key in methods) {
                if (!(key in proto)) _redefine(proto, key, methods[key]);
            } else _export(_export.P + _export.F * (BUGGY || VALUES_BUG), NAME, methods);
        }
        return methods;
    };

    var addToUnscopables = ( _addToUnscopables$1 && _addToUnscopables ) || _addToUnscopables$1;

    // 22.1.3.4 Array.prototype.entries()
    // 22.1.3.13 Array.prototype.keys()
    // 22.1.3.29 Array.prototype.values()
    // 22.1.3.30 Array.prototype[@@iterator]()
    var es6_array_iterator = _iterDefine(Array, 'Array', function (iterated, kind) {
        this._t = _toIobject(iterated); // target
        this._i = 0;                   // next index
        this._k = kind;                // kind
        // 22.1.5.2.1 %ArrayIteratorPrototype%.next()
    }, function () {
        var O = this._t;
        var kind = this._k;
        var index = this._i++;
        if (!O || index >= O.length) {
            this._t = undefined;
            return _iterStep(1);
        }
        if (kind == 'keys') return _iterStep(0, index);
        if (kind == 'values') return _iterStep(0, O[index]);
        return _iterStep(0, [index, O[index]]);
    }, 'values');

    // argumentsList[@@iterator] is %ArrayProto_values% (9.4.4.6, 9.4.4.7)
    _iterators.Arguments = _iterators.Array;

    addToUnscopables('keys');
    addToUnscopables('values');
    addToUnscopables('entries');

    var ITERATOR$1 = _wks('iterator');
    var TO_STRING_TAG = _wks('toStringTag');
    var ArrayValues = _iterators.Array;

    var DOMIterables = {
        CSSRuleList: true, // TODO: Not spec compliant, should be false.
        CSSStyleDeclaration: false,
        CSSValueList: false,
        ClientRectList: false,
        DOMRectList: false,
        DOMStringList: false,
        DOMTokenList: true,
        DataTransferItemList: false,
        FileList: false,
        HTMLAllCollection: false,
        HTMLCollection: false,
        HTMLFormElement: false,
        HTMLSelectElement: false,
        MediaList: true, // TODO: Not spec compliant, should be false.
        MimeTypeArray: false,
        NamedNodeMap: false,
        NodeList: true,
        PaintRequestList: false,
        Plugin: false,
        PluginArray: false,
        SVGLengthList: false,
        SVGNumberList: false,
        SVGPathSegList: false,
        SVGPointList: false,
        SVGStringList: false,
        SVGTransformList: false,
        SourceBufferList: false,
        StyleSheetList: true, // TODO: Not spec compliant, should be false.
        TextTrackCueList: false,
        TextTrackList: false,
        TouchList: false
    };

    for (var collections = _objectKeys(DOMIterables), i = 0; i < collections.length; i++) {
        var NAME = collections[i];
        var explicit = DOMIterables[NAME];
        var Collection = _global[NAME];
        var proto = Collection && Collection.prototype;
        var key;
        if (proto) {
            if (!proto[ITERATOR$1]) _hide(proto, ITERATOR$1, ArrayValues);
            if (!proto[TO_STRING_TAG]) _hide(proto, TO_STRING_TAG, NAME);
            _iterators[NAME] = ArrayValues;
            if (explicit) for (key in es6_array_iterator) if (!proto[key]) _redefine(proto, key, es6_array_iterator[key], true);
        }
    }

    function _classCallCheck(instance, Constructor) {
        if (!(instance instanceof Constructor)) {
            throw new TypeError("Cannot call a class as a function");
        }
    }

    var classCallCheck = _classCallCheck;

    function _defineProperties(target, props) {
        for (var i = 0; i < props.length; i++) {
            var descriptor = props[i];
            descriptor.enumerable = descriptor.enumerable || false;
            descriptor.configurable = true;
            if ("value" in descriptor) descriptor.writable = true;
            Object.defineProperty(target, descriptor.key, descriptor);
        }
    }

    function _createClass(Constructor, protoProps, staticProps) {
        if (protoProps) _defineProperties(Constructor.prototype, protoProps);
        if (staticProps) _defineProperties(Constructor, staticProps);
        return Constructor;
    }

    var createClass = _createClass;

    // getting tag from 19.1.3.6 Object.prototype.toString()

    var TAG$1 = _wks('toStringTag');
    // ES3 wrong here
    var ARG = _cof(function () { return arguments; }()) == 'Arguments';

    // fallback for IE11 Script Access Denied error
    var tryGet = function (it, key) {
        try {
            return it[key];
        } catch (e) { /* empty */ }
    };

    var _classof = function (it) {
        var O, T, B;
        return it === undefined ? 'Undefined' : it === null ? 'Null'
            // @@toStringTag case
            : typeof (T = tryGet(O = Object(it), TAG$1)) == 'string' ? T
                // builtinTag case
                : ARG ? _cof(O)
                    // ES3 arguments fallback
                    : (B = _cof(O)) == 'Object' && typeof O.callee == 'function' ? 'Arguments' : B;
    };

    var _anInstance = function (it, Constructor, name, forbiddenField) {
        if (!(it instanceof Constructor) || (forbiddenField !== undefined && forbiddenField in it)) {
            throw TypeError(name + ': incorrect invocation!');
        } return it;
    };

    // call something on iterator step with safe closing on error

    var _iterCall = function (iterator, fn, value, entries) {
        try {
            return entries ? fn(_anObject(value)[0], value[1]) : fn(value);
            // 7.4.6 IteratorClose(iterator, completion)
        } catch (e) {
            var ret = iterator['return'];
            if (ret !== undefined) _anObject(ret.call(iterator));
            throw e;
        }
    };

    // check on default Array iterator

    var ITERATOR$2 = _wks('iterator');
    var ArrayProto$1 = Array.prototype;

    var _isArrayIter = function (it) {
        return it !== undefined && (_iterators.Array === it || ArrayProto$1[ITERATOR$2] === it);
    };

    var ITERATOR$3 = _wks('iterator');

    var core_getIteratorMethod = _core.getIteratorMethod = function (it) {
        if (it != undefined) return it[ITERATOR$3]
            || it['@@iterator']
            || _iterators[_classof(it)];
    };

    var _forOf = createCommonjsModule(function (module) {
        var BREAK = {};
        var RETURN = {};
        var exports = module.exports = function (iterable, entries, fn, that, ITERATOR) {
            var iterFn = ITERATOR ? function () { return iterable; } : core_getIteratorMethod(iterable);
            var f = _ctx(fn, that, entries ? 2 : 1);
            var index = 0;
            var length, step, iterator, result;
            if (typeof iterFn != 'function') throw TypeError(iterable + ' is not iterable!');
            // fast case for arrays with default iterator
            if (_isArrayIter(iterFn)) for (length = _toLength(iterable.length); length > index; index++) {
                result = entries ? f(_anObject(step = iterable[index])[0], step[1]) : f(iterable[index]);
                if (result === BREAK || result === RETURN) return result;
            } else for (iterator = iterFn.call(iterable); !(step = iterator.next()).done;) {
                result = _iterCall(iterator, f, step.value, entries);
                if (result === BREAK || result === RETURN) return result;
            }
        };
        exports.BREAK = BREAK;
        exports.RETURN = RETURN;
    });

    // 7.3.20 SpeciesConstructor(O, defaultConstructor)


    var SPECIES = _wks('species');
    var _speciesConstructor = function (O, D) {
        var C = _anObject(O).constructor;
        var S;
        return C === undefined || (S = _anObject(C)[SPECIES]) == undefined ? D : _aFunction(S);
    };

    // fast apply, http://jsperf.lnkit.com/fast-apply/5
    var _invoke = function (fn, args, that) {
        var un = that === undefined;
        switch (args.length) {
            case 0: return un ? fn()
                : fn.call(that);
            case 1: return un ? fn(args[0])
                : fn.call(that, args[0]);
            case 2: return un ? fn(args[0], args[1])
                : fn.call(that, args[0], args[1]);
            case 3: return un ? fn(args[0], args[1], args[2])
                : fn.call(that, args[0], args[1], args[2]);
            case 4: return un ? fn(args[0], args[1], args[2], args[3])
                : fn.call(that, args[0], args[1], args[2], args[3]);
        } return fn.apply(that, args);
    };

    var process$1 = _global.process;
    var setTask = _global.setImmediate;
    var clearTask = _global.clearImmediate;
    var MessageChannel = _global.MessageChannel;
    var Dispatch = _global.Dispatch;
    var counter = 0;
    var queue = {};
    var ONREADYSTATECHANGE = 'onreadystatechange';
    var defer, channel, port;
    var run = function () {
        var id = +this;
        // eslint-disable-next-line no-prototype-builtins
        if (queue.hasOwnProperty(id)) {
            var fn = queue[id];
            delete queue[id];
            fn();
        }
    };
    var listener = function (event) {
        run.call(event.data);
    };
    // Node.js 0.9+ & IE10+ has setImmediate, otherwise:
    if (!setTask || !clearTask) {
        setTask = function setImmediate(fn) {
            var args = [];
            var i = 1;
            while (arguments.length > i) args.push(arguments[i++]);
            queue[++counter] = function () {
                // eslint-disable-next-line no-new-func
                _invoke(typeof fn == 'function' ? fn : Function(fn), args);
            };
            defer(counter);
            return counter;
        };
        clearTask = function clearImmediate(id) {
            delete queue[id];
        };
        // Node.js 0.8-
        if (_cof(process$1) == 'process') {
            defer = function (id) {
                process$1.nextTick(_ctx(run, id, 1));
            };
            // Sphere (JS game engine) Dispatch API
        } else if (Dispatch && Dispatch.now) {
            defer = function (id) {
                Dispatch.now(_ctx(run, id, 1));
            };
            // Browsers with MessageChannel, includes WebWorkers
        } else if (MessageChannel) {
            channel = new MessageChannel();
            port = channel.port2;
            channel.port1.onmessage = listener;
            defer = _ctx(port.postMessage, port, 1);
            // Browsers with postMessage, skip WebWorkers
            // IE8 has postMessage, but it's sync & typeof its postMessage is 'object'
        } else if (_global.addEventListener && typeof postMessage == 'function' && !_global.importScripts) {
            defer = function (id) {
                _global.postMessage(id + '', '*');
            };
            _global.addEventListener('message', listener, false);
            // IE8-
        } else if (ONREADYSTATECHANGE in _domCreate('script')) {
            defer = function (id) {
                _html.appendChild(_domCreate('script'))[ONREADYSTATECHANGE] = function () {
                    _html.removeChild(this);
                    run.call(id);
                };
            };
            // Rest old browsers
        } else {
            defer = function (id) {
                setTimeout(_ctx(run, id, 1), 0);
            };
        }
    }
    var _task = {
        set: setTask,
        clear: clearTask
    };

    var macrotask = _task.set;
    var Observer = _global.MutationObserver || _global.WebKitMutationObserver;
    var process$2 = _global.process;
    var Promise$1 = _global.Promise;
    var isNode = _cof(process$2) == 'process';

    var _microtask = function () {
        var head, last, notify;

        var flush = function () {
            var parent, fn;
            if (isNode && (parent = process$2.domain)) parent.exit();
            while (head) {
                fn = head.fn;
                head = head.next;
                try {
                    fn();
                } catch (e) {
                    if (head) notify();
                    else last = undefined;
                    throw e;
                }
            } last = undefined;
            if (parent) parent.enter();
        };

        // Node.js
        if (isNode) {
            notify = function () {
                process$2.nextTick(flush);
            };
            // browsers with MutationObserver, except iOS Safari - https://github.com/zloirock/core-js/issues/339
        } else if (Observer && !(_global.navigator && _global.navigator.standalone)) {
            var toggle = true;
            var node = document.createTextNode('');
            new Observer(flush).observe(node, { characterData: true }); // eslint-disable-line no-new
            notify = function () {
                node.data = toggle = !toggle;
            };
            // environments with maybe non-completely correct, but existent Promise
        } else if (Promise$1 && Promise$1.resolve) {
            // Promise.resolve without an argument throws an error in LG WebOS 2
            var promise = Promise$1.resolve(undefined);
            notify = function () {
                promise.then(flush);
            };
            // for other environments - macrotask based on:
            // - setImmediate
            // - MessageChannel
            // - window.postMessag
            // - onreadystatechange
            // - setTimeout
        } else {
            notify = function () {
                // strange IE + webpack dev server bug - use .call(global)
                macrotask.call(_global, flush);
            };
        }

        return function (fn) {
            var task = { fn: fn, next: undefined };
            if (last) last.next = task;
            if (!head) {
                head = task;
                notify();
            } last = task;
        };
    };

    // 25.4.1.5 NewPromiseCapability(C)


    function PromiseCapability(C) {
        var resolve, reject;
        this.promise = new C(function ($$resolve, $$reject) {
            if (resolve !== undefined || reject !== undefined) throw TypeError('Bad Promise constructor');
            resolve = $$resolve;
            reject = $$reject;
        });
        this.resolve = _aFunction(resolve);
        this.reject = _aFunction(reject);
    }

    var f$1 = function (C) {
        return new PromiseCapability(C);
    };

    var _newPromiseCapability = {
        f: f$1
    };

    var _perform = function (exec) {
        try {
            return { e: false, v: exec() };
        } catch (e) {
            return { e: true, v: e };
        }
    };

    var navigator$1 = _global.navigator;

    var _userAgent = navigator$1 && navigator$1.userAgent || '';

    var _promiseResolve = function (C, x) {
        _anObject(C);
        if (_isObject(x) && x.constructor === C) return x;
        var promiseCapability = _newPromiseCapability.f(C);
        var resolve = promiseCapability.resolve;
        resolve(x);
        return promiseCapability.promise;
    };

    var _redefineAll = function (target, src, safe) {
        for (var key in src) _redefine(target, key, src[key], safe);
        return target;
    };

    var SPECIES$1 = _wks('species');

    var _setSpecies = function (KEY) {
        var C = _global[KEY];
        if (_descriptors && C && !C[SPECIES$1]) _objectDp.f(C, SPECIES$1, {
            configurable: true,
            get: function () { return this; }
        });
    };

    var ITERATOR$4 = _wks('iterator');
    var SAFE_CLOSING = false;

    try {
        var riter = [7][ITERATOR$4]();
        riter['return'] = function () { SAFE_CLOSING = true; };
    } catch (e) { /* empty */ }

    var _iterDetect = function (exec, skipClosing) {
        if (!skipClosing && !SAFE_CLOSING) return false;
        var safe = false;
        try {
            var arr = [7];
            var iter = arr[ITERATOR$4]();
            iter.next = function () { return { done: safe = true }; };
            arr[ITERATOR$4] = function () { return iter; };
            exec(arr);
        } catch (e) { /* empty */ }
        return safe;
    };

    var task = _task.set;
    var microtask = _microtask();




    var PROMISE = 'Promise';
    var TypeError$1 = _global.TypeError;
    var process$3 = _global.process;
    var versions = process$3 && process$3.versions;
    var v8 = versions && versions.v8 || '';
    var $Promise = _global[PROMISE];
    var isNode$1 = _classof(process$3) == 'process';
    var empty = function () { /* empty */ };
    var Internal, newGenericPromiseCapability, OwnPromiseCapability, Wrapper;
    var newPromiseCapability = newGenericPromiseCapability = _newPromiseCapability.f;

    var USE_NATIVE = !!function () {
        try {
            // correct subclassing with @@species support
            var promise = $Promise.resolve(1);
            var FakePromise = (promise.constructor = {})[_wks('species')] = function (exec) {
                exec(empty, empty);
            };
            // unhandled rejections tracking support, NodeJS Promise without it fails @@species test
            return (isNode$1 || typeof PromiseRejectionEvent == 'function')
                && promise.then(empty) instanceof FakePromise
                // v8 6.6 (Node 10 and Chrome 66) have a bug with resolving custom thenables
                // https://bugs.chromium.org/p/chromium/issues/detail?id=830565
                // we can't detect it synchronously, so just check versions
                && v8.indexOf('6.6') !== 0
                && _userAgent.indexOf('Chrome/66') === -1;
        } catch (e) { /* empty */ }
    }();

    // helpers
    var isThenable = function (it) {
        var then;
        return _isObject(it) && typeof (then = it.then) == 'function' ? then : false;
    };
    var notify = function (promise, isReject) {
        if (promise._n) return;
        promise._n = true;
        var chain = promise._c;
        microtask(function () {
            var value = promise._v;
            var ok = promise._s == 1;
            var i = 0;
            var run = function (reaction) {
                var handler = ok ? reaction.ok : reaction.fail;
                var resolve = reaction.resolve;
                var reject = reaction.reject;
                var domain = reaction.domain;
                var result, then, exited;
                try {
                    if (handler) {
                        if (!ok) {
                            if (promise._h == 2) onHandleUnhandled(promise);
                            promise._h = 1;
                        }
                        if (handler === true) result = value;
                        else {
                            if (domain) domain.enter();
                            result = handler(value); // may throw
                            if (domain) {
                                domain.exit();
                                exited = true;
                            }
                        }
                        if (result === reaction.promise) {
                            reject(TypeError$1('Promise-chain cycle'));
                        } else if (then = isThenable(result)) {
                            then.call(result, resolve, reject);
                        } else resolve(result);
                    } else reject(value);
                } catch (e) {
                    if (domain && !exited) domain.exit();
                    reject(e);
                }
            };
            while (chain.length > i) run(chain[i++]); // variable length - can't use forEach
            promise._c = [];
            promise._n = false;
            if (isReject && !promise._h) onUnhandled(promise);
        });
    };
    var onUnhandled = function (promise) {
        task.call(_global, function () {
            var value = promise._v;
            var unhandled = isUnhandled(promise);
            var result, handler, console;
            if (unhandled) {
                result = _perform(function () {
                    if (isNode$1) {
                        process$3.emit('unhandledRejection', value, promise);
                    } else if (handler = _global.onunhandledrejection) {
                        handler({ promise: promise, reason: value });
                    } else if ((console = _global.console) && console.error) {
                        console.error('Unhandled promise rejection', value);
                    }
                });
                // Browsers should not trigger `rejectionHandled` event if it was handled here, NodeJS - should
                promise._h = isNode$1 || isUnhandled(promise) ? 2 : 1;
            } promise._a = undefined;
            if (unhandled && result.e) throw result.v;
        });
    };
    var isUnhandled = function (promise) {
        return promise._h !== 1 && (promise._a || promise._c).length === 0;
    };
    var onHandleUnhandled = function (promise) {
        task.call(_global, function () {
            var handler;
            if (isNode$1) {
                process$3.emit('rejectionHandled', promise);
            } else if (handler = _global.onrejectionhandled) {
                handler({ promise: promise, reason: promise._v });
            }
        });
    };
    var $reject = function (value) {
        var promise = this;
        if (promise._d) return;
        promise._d = true;
        promise = promise._w || promise; // unwrap
        promise._v = value;
        promise._s = 2;
        if (!promise._a) promise._a = promise._c.slice();
        notify(promise, true);
    };
    var $resolve = function (value) {
        var promise = this;
        var then;
        if (promise._d) return;
        promise._d = true;
        promise = promise._w || promise; // unwrap
        try {
            if (promise === value) throw TypeError$1("Promise can't be resolved itself");
            if (then = isThenable(value)) {
                microtask(function () {
                    var wrapper = { _w: promise, _d: false }; // wrap
                    try {
                        then.call(value, _ctx($resolve, wrapper, 1), _ctx($reject, wrapper, 1));
                    } catch (e) {
                        $reject.call(wrapper, e);
                    }
                });
            } else {
                promise._v = value;
                promise._s = 1;
                notify(promise, false);
            }
        } catch (e) {
            $reject.call({ _w: promise, _d: false }, e); // wrap
        }
    };

    // constructor polyfill
    if (!USE_NATIVE) {
        // 25.4.3.1 Promise(executor)
        $Promise = function Promise(executor) {
            _anInstance(this, $Promise, PROMISE, '_h');
            _aFunction(executor);
            Internal.call(this);
            try {
                executor(_ctx($resolve, this, 1), _ctx($reject, this, 1));
            } catch (err) {
                $reject.call(this, err);
            }
        };
        // eslint-disable-next-line no-unused-vars
        Internal = function Promise(executor) {
            this._c = [];             // <- awaiting reactions
            this._a = undefined;      // <- checked in isUnhandled reactions
            this._s = 0;              // <- state
            this._d = false;          // <- done
            this._v = undefined;      // <- value
            this._h = 0;              // <- rejection state, 0 - default, 1 - handled, 2 - unhandled
            this._n = false;          // <- notify
        };
        Internal.prototype = _redefineAll($Promise.prototype, {
            // 25.4.5.3 Promise.prototype.then(onFulfilled, onRejected)
            then: function then(onFulfilled, onRejected) {
                var reaction = newPromiseCapability(_speciesConstructor(this, $Promise));
                reaction.ok = typeof onFulfilled == 'function' ? onFulfilled : true;
                reaction.fail = typeof onRejected == 'function' && onRejected;
                reaction.domain = isNode$1 ? process$3.domain : undefined;
                this._c.push(reaction);
                if (this._a) this._a.push(reaction);
                if (this._s) notify(this, false);
                return reaction.promise;
            },
            // 25.4.5.1 Promise.prototype.catch(onRejected)
            'catch': function (onRejected) {
                return this.then(undefined, onRejected);
            }
        });
        OwnPromiseCapability = function () {
            var promise = new Internal();
            this.promise = promise;
            this.resolve = _ctx($resolve, promise, 1);
            this.reject = _ctx($reject, promise, 1);
        };
        _newPromiseCapability.f = newPromiseCapability = function (C) {
            return C === $Promise || C === Wrapper
                ? new OwnPromiseCapability(C)
                : newGenericPromiseCapability(C);
        };
    }

    _export(_export.G + _export.W + _export.F * !USE_NATIVE, { Promise: $Promise });
    _setToStringTag($Promise, PROMISE);
    _setSpecies(PROMISE);
    Wrapper = _core[PROMISE];

    // statics
    _export(_export.S + _export.F * !USE_NATIVE, PROMISE, {
        // 25.4.4.5 Promise.reject(r)
        reject: function reject(r) {
            var capability = newPromiseCapability(this);
            var $$reject = capability.reject;
            $$reject(r);
            return capability.promise;
        }
    });
    _export(_export.S + _export.F * (_library || !USE_NATIVE), PROMISE, {
        // 25.4.4.6 Promise.resolve(x)
        resolve: function resolve(x) {
            return _promiseResolve(_library && this === Wrapper ? $Promise : this, x);
        }
    });
    _export(_export.S + _export.F * !(USE_NATIVE && _iterDetect(function (iter) {
        $Promise.all(iter)['catch'](empty);
    })), PROMISE, {
        // 25.4.4.1 Promise.all(iterable)
        all: function all(iterable) {
            var C = this;
            var capability = newPromiseCapability(C);
            var resolve = capability.resolve;
            var reject = capability.reject;
            var result = _perform(function () {
                var values = [];
                var index = 0;
                var remaining = 1;
                _forOf(iterable, false, function (promise) {
                    var $index = index++;
                    var alreadyCalled = false;
                    values.push(undefined);
                    remaining++;
                    C.resolve(promise).then(function (value) {
                        if (alreadyCalled) return;
                        alreadyCalled = true;
                        values[$index] = value;
                        --remaining || resolve(values);
                    }, reject);
                });
                --remaining || resolve(values);
            });
            if (result.e) reject(result.v);
            return capability.promise;
        },
        // 25.4.4.4 Promise.race(iterable)
        race: function race(iterable) {
            var C = this;
            var capability = newPromiseCapability(C);
            var reject = capability.reject;
            var result = _perform(function () {
                _forOf(iterable, false, function (promise) {
                    C.resolve(promise).then(capability.resolve, reject);
                });
            });
            if (result.e) reject(result.v);
            return capability.promise;
        }
    });

    var runtime = createCommonjsModule(function (module) {
        /**
         * Copyright (c) 2014-present, Facebook, Inc.
         *
         * This source code is licensed under the MIT license found in the
         * LICENSE file in the root directory of this source tree.
         */

        !(function(global) {

            var Op = Object.prototype;
            var hasOwn = Op.hasOwnProperty;
            var undefined; // More compressible than void 0.
            var $Symbol = typeof Symbol === "function" ? Symbol : {};
            var iteratorSymbol = $Symbol.iterator || "@@iterator";
            var asyncIteratorSymbol = $Symbol.asyncIterator || "@@asyncIterator";
            var toStringTagSymbol = $Symbol.toStringTag || "@@toStringTag";
            var runtime = global.regeneratorRuntime;
            if (runtime) {
                {
                    // If regeneratorRuntime is defined globally and we're in a module,
                    // make the exports object identical to regeneratorRuntime.
                    module.exports = runtime;
                }
                // Don't bother evaluating the rest of this file if the runtime was
                // already defined globally.
                return;
            }

            // Define the runtime globally (as expected by generated code) as either
            // module.exports (if we're in a module) or a new, empty object.
            runtime = global.regeneratorRuntime = module.exports;

            function wrap(innerFn, outerFn, self, tryLocsList) {
                // If outerFn provided and outerFn.prototype is a Generator, then outerFn.prototype instanceof Generator.
                var protoGenerator = outerFn && outerFn.prototype instanceof Generator ? outerFn : Generator;
                var generator = Object.create(protoGenerator.prototype);
                var context = new Context(tryLocsList || []);

                // The ._invoke method unifies the implementations of the .next,
                // .throw, and .return methods.
                generator._invoke = makeInvokeMethod(innerFn, self, context);

                return generator;
            }
            runtime.wrap = wrap;

            // Try/catch helper to minimize deoptimizations. Returns a completion
            // record like context.tryEntries[i].completion. This interface could
            // have been (and was previously) designed to take a closure to be
            // invoked without arguments, but in all the cases we care about we
            // already have an existing method we want to call, so there's no need
            // to create a new function object. We can even get away with assuming
            // the method takes exactly one argument, since that happens to be true
            // in every case, so we don't have to touch the arguments object. The
            // only additional allocation required is the completion record, which
            // has a stable shape and so hopefully should be cheap to allocate.
            function tryCatch(fn, obj, arg) {
                try {
                    return { type: "normal", arg: fn.call(obj, arg) };
                } catch (err) {
                    return { type: "throw", arg: err };
                }
            }

            var GenStateSuspendedStart = "suspendedStart";
            var GenStateSuspendedYield = "suspendedYield";
            var GenStateExecuting = "executing";
            var GenStateCompleted = "completed";

            // Returning this object from the innerFn has the same effect as
            // breaking out of the dispatch switch statement.
            var ContinueSentinel = {};

            // Dummy constructor functions that we use as the .constructor and
            // .constructor.prototype properties for functions that return Generator
            // objects. For full spec compliance, you may wish to configure your
            // minifier not to mangle the names of these two functions.
            function Generator() {}
            function GeneratorFunction() {}
            function GeneratorFunctionPrototype() {}

            // This is a polyfill for %IteratorPrototype% for environments that
            // don't natively support it.
            var IteratorPrototype = {};
            IteratorPrototype[iteratorSymbol] = function () {
                return this;
            };

            var getProto = Object.getPrototypeOf;
            var NativeIteratorPrototype = getProto && getProto(getProto(values([])));
            if (NativeIteratorPrototype &&
                NativeIteratorPrototype !== Op &&
                hasOwn.call(NativeIteratorPrototype, iteratorSymbol)) {
                // This environment has a native %IteratorPrototype%; use it instead
                // of the polyfill.
                IteratorPrototype = NativeIteratorPrototype;
            }

            var Gp = GeneratorFunctionPrototype.prototype =
                Generator.prototype = Object.create(IteratorPrototype);
            GeneratorFunction.prototype = Gp.constructor = GeneratorFunctionPrototype;
            GeneratorFunctionPrototype.constructor = GeneratorFunction;
            GeneratorFunctionPrototype[toStringTagSymbol] =
                GeneratorFunction.displayName = "GeneratorFunction";

            // Helper for defining the .next, .throw, and .return methods of the
            // Iterator interface in terms of a single ._invoke method.
            function defineIteratorMethods(prototype) {
                ["next", "throw", "return"].forEach(function(method) {
                    prototype[method] = function(arg) {
                        return this._invoke(method, arg);
                    };
                });
            }

            runtime.isGeneratorFunction = function(genFun) {
                var ctor = typeof genFun === "function" && genFun.constructor;
                return ctor
                    ? ctor === GeneratorFunction ||
                    // For the native GeneratorFunction constructor, the best we can
                    // do is to check its .name property.
                    (ctor.displayName || ctor.name) === "GeneratorFunction"
                    : false;
            };

            runtime.mark = function(genFun) {
                if (Object.setPrototypeOf) {
                    Object.setPrototypeOf(genFun, GeneratorFunctionPrototype);
                } else {
                    genFun.__proto__ = GeneratorFunctionPrototype;
                    if (!(toStringTagSymbol in genFun)) {
                        genFun[toStringTagSymbol] = "GeneratorFunction";
                    }
                }
                genFun.prototype = Object.create(Gp);
                return genFun;
            };

            // Within the body of any async function, `await x` is transformed to
            // `yield regeneratorRuntime.awrap(x)`, so that the runtime can test
            // `hasOwn.call(value, "__await")` to determine if the yielded value is
            // meant to be awaited.
            runtime.awrap = function(arg) {
                return { __await: arg };
            };

            function AsyncIterator(generator) {
                function invoke(method, arg, resolve, reject) {
                    var record = tryCatch(generator[method], generator, arg);
                    if (record.type === "throw") {
                        reject(record.arg);
                    } else {
                        var result = record.arg;
                        var value = result.value;
                        if (value &&
                            typeof value === "object" &&
                            hasOwn.call(value, "__await")) {
                            return Promise.resolve(value.__await).then(function(value) {
                                invoke("next", value, resolve, reject);
                            }, function(err) {
                                invoke("throw", err, resolve, reject);
                            });
                        }

                        return Promise.resolve(value).then(function(unwrapped) {
                            // When a yielded Promise is resolved, its final value becomes
                            // the .value of the Promise<{value,done}> result for the
                            // current iteration.
                            result.value = unwrapped;
                            resolve(result);
                        }, function(error) {
                            // If a rejected Promise was yielded, throw the rejection back
                            // into the async generator function so it can be handled there.
                            return invoke("throw", error, resolve, reject);
                        });
                    }
                }

                var previousPromise;

                function enqueue(method, arg) {
                    function callInvokeWithMethodAndArg() {
                        return new Promise(function(resolve, reject) {
                            invoke(method, arg, resolve, reject);
                        });
                    }

                    return previousPromise =
                        // If enqueue has been called before, then we want to wait until
                        // all previous Promises have been resolved before calling invoke,
                        // so that results are always delivered in the correct order. If
                        // enqueue has not been called before, then it is important to
                        // call invoke immediately, without waiting on a callback to fire,
                        // so that the async generator function has the opportunity to do
                        // any necessary setup in a predictable way. This predictability
                        // is why the Promise constructor synchronously invokes its
                        // executor callback, and why async functions synchronously
                        // execute code before the first await. Since we implement simple
                        // async functions in terms of async generators, it is especially
                        // important to get this right, even though it requires care.
                        previousPromise ? previousPromise.then(
                            callInvokeWithMethodAndArg,
                            // Avoid propagating failures to Promises returned by later
                            // invocations of the iterator.
                            callInvokeWithMethodAndArg
                        ) : callInvokeWithMethodAndArg();
                }

                // Define the unified helper method that is used to implement .next,
                // .throw, and .return (see defineIteratorMethods).
                this._invoke = enqueue;
            }

            defineIteratorMethods(AsyncIterator.prototype);
            AsyncIterator.prototype[asyncIteratorSymbol] = function () {
                return this;
            };
            runtime.AsyncIterator = AsyncIterator;

            // Note that simple async functions are implemented on top of
            // AsyncIterator objects; they just return a Promise for the value of
            // the final result produced by the iterator.
            runtime.async = function(innerFn, outerFn, self, tryLocsList) {
                var iter = new AsyncIterator(
                    wrap(innerFn, outerFn, self, tryLocsList)
                );

                return runtime.isGeneratorFunction(outerFn)
                    ? iter // If outerFn is a generator, return the full iterator.
                    : iter.next().then(function(result) {
                        return result.done ? result.value : iter.next();
                    });
            };

            function makeInvokeMethod(innerFn, self, context) {
                var state = GenStateSuspendedStart;

                return function invoke(method, arg) {
                    if (state === GenStateExecuting) {
                        throw new Error("Generator is already running");
                    }

                    if (state === GenStateCompleted) {
                        if (method === "throw") {
                            throw arg;
                        }

                        // Be forgiving, per 25.3.3.3.3 of the spec:
                        // https://people.mozilla.org/~jorendorff/es6-draft.html#sec-generatorresume
                        return doneResult();
                    }

                    context.method = method;
                    context.arg = arg;

                    while (true) {
                        var delegate = context.delegate;
                        if (delegate) {
                            var delegateResult = maybeInvokeDelegate(delegate, context);
                            if (delegateResult) {
                                if (delegateResult === ContinueSentinel) continue;
                                return delegateResult;
                            }
                        }

                        if (context.method === "next") {
                            // Setting context._sent for legacy support of Babel's
                            // function.sent implementation.
                            context.sent = context._sent = context.arg;

                        } else if (context.method === "throw") {
                            if (state === GenStateSuspendedStart) {
                                state = GenStateCompleted;
                                throw context.arg;
                            }

                            context.dispatchException(context.arg);

                        } else if (context.method === "return") {
                            context.abrupt("return", context.arg);
                        }

                        state = GenStateExecuting;

                        var record = tryCatch(innerFn, self, context);
                        if (record.type === "normal") {
                            // If an exception is thrown from innerFn, we leave state ===
                            // GenStateExecuting and loop back for another invocation.
                            state = context.done
                                ? GenStateCompleted
                                : GenStateSuspendedYield;

                            if (record.arg === ContinueSentinel) {
                                continue;
                            }

                            return {
                                value: record.arg,
                                done: context.done
                            };

                        } else if (record.type === "throw") {
                            state = GenStateCompleted;
                            // Dispatch the exception by looping back around to the
                            // context.dispatchException(context.arg) call above.
                            context.method = "throw";
                            context.arg = record.arg;
                        }
                    }
                };
            }

            // Call delegate.iterator[context.method](context.arg) and handle the
            // result, either by returning a { value, done } result from the
            // delegate iterator, or by modifying context.method and context.arg,
            // setting context.delegate to null, and returning the ContinueSentinel.
            function maybeInvokeDelegate(delegate, context) {
                var method = delegate.iterator[context.method];
                if (method === undefined) {
                    // A .throw or .return when the delegate iterator has no .throw
                    // method always terminates the yield* loop.
                    context.delegate = null;

                    if (context.method === "throw") {
                        if (delegate.iterator.return) {
                            // If the delegate iterator has a return method, give it a
                            // chance to clean up.
                            context.method = "return";
                            context.arg = undefined;
                            maybeInvokeDelegate(delegate, context);

                            if (context.method === "throw") {
                                // If maybeInvokeDelegate(context) changed context.method from
                                // "return" to "throw", let that override the TypeError below.
                                return ContinueSentinel;
                            }
                        }

                        context.method = "throw";
                        context.arg = new TypeError(
                            "The iterator does not provide a 'throw' method");
                    }

                    return ContinueSentinel;
                }

                var record = tryCatch(method, delegate.iterator, context.arg);

                if (record.type === "throw") {
                    context.method = "throw";
                    context.arg = record.arg;
                    context.delegate = null;
                    return ContinueSentinel;
                }

                var info = record.arg;

                if (! info) {
                    context.method = "throw";
                    context.arg = new TypeError("iterator result is not an object");
                    context.delegate = null;
                    return ContinueSentinel;
                }

                if (info.done) {
                    // Assign the result of the finished delegate to the temporary
                    // variable specified by delegate.resultName (see delegateYield).
                    context[delegate.resultName] = info.value;

                    // Resume execution at the desired location (see delegateYield).
                    context.next = delegate.nextLoc;

                    // If context.method was "throw" but the delegate handled the
                    // exception, let the outer generator proceed normally. If
                    // context.method was "next", forget context.arg since it has been
                    // "consumed" by the delegate iterator. If context.method was
                    // "return", allow the original .return call to continue in the
                    // outer generator.
                    if (context.method !== "return") {
                        context.method = "next";
                        context.arg = undefined;
                    }

                } else {
                    // Re-yield the result returned by the delegate method.
                    return info;
                }

                // The delegate iterator is finished, so forget it and continue with
                // the outer generator.
                context.delegate = null;
                return ContinueSentinel;
            }

            // Define Generator.prototype.{next,throw,return} in terms of the
            // unified ._invoke helper method.
            defineIteratorMethods(Gp);

            Gp[toStringTagSymbol] = "Generator";

            // A Generator should always return itself as the iterator object when the
            // @@iterator function is called on it. Some browsers' implementations of the
            // iterator prototype chain incorrectly implement this, causing the Generator
            // object to not be returned from this call. This ensures that doesn't happen.
            // See https://github.com/facebook/regenerator/issues/274 for more details.
            Gp[iteratorSymbol] = function() {
                return this;
            };

            Gp.toString = function() {
                return "[object Generator]";
            };

            function pushTryEntry(locs) {
                var entry = { tryLoc: locs[0] };

                if (1 in locs) {
                    entry.catchLoc = locs[1];
                }

                if (2 in locs) {
                    entry.finallyLoc = locs[2];
                    entry.afterLoc = locs[3];
                }

                this.tryEntries.push(entry);
            }

            function resetTryEntry(entry) {
                var record = entry.completion || {};
                record.type = "normal";
                delete record.arg;
                entry.completion = record;
            }

            function Context(tryLocsList) {
                // The root entry object (effectively a try statement without a catch
                // or a finally block) gives us a place to store values thrown from
                // locations where there is no enclosing try statement.
                this.tryEntries = [{ tryLoc: "root" }];
                tryLocsList.forEach(pushTryEntry, this);
                this.reset(true);
            }

            runtime.keys = function(object) {
                var keys = [];
                for (var key in object) {
                    keys.push(key);
                }
                keys.reverse();

                // Rather than returning an object with a next method, we keep
                // things simple and return the next function itself.
                return function next() {
                    while (keys.length) {
                        var key = keys.pop();
                        if (key in object) {
                            next.value = key;
                            next.done = false;
                            return next;
                        }
                    }

                    // To avoid creating an additional object, we just hang the .value
                    // and .done properties off the next function object itself. This
                    // also ensures that the minifier will not anonymize the function.
                    next.done = true;
                    return next;
                };
            };

            function values(iterable) {
                if (iterable) {
                    var iteratorMethod = iterable[iteratorSymbol];
                    if (iteratorMethod) {
                        return iteratorMethod.call(iterable);
                    }

                    if (typeof iterable.next === "function") {
                        return iterable;
                    }

                    if (!isNaN(iterable.length)) {
                        var i = -1, next = function next() {
                            while (++i < iterable.length) {
                                if (hasOwn.call(iterable, i)) {
                                    next.value = iterable[i];
                                    next.done = false;
                                    return next;
                                }
                            }

                            next.value = undefined;
                            next.done = true;

                            return next;
                        };

                        return next.next = next;
                    }
                }

                // Return an iterator with no values.
                return { next: doneResult };
            }
            runtime.values = values;

            function doneResult() {
                return { value: undefined, done: true };
            }

            Context.prototype = {
                constructor: Context,

                reset: function(skipTempReset) {
                    this.prev = 0;
                    this.next = 0;
                    // Resetting context._sent for legacy support of Babel's
                    // function.sent implementation.
                    this.sent = this._sent = undefined;
                    this.done = false;
                    this.delegate = null;

                    this.method = "next";
                    this.arg = undefined;

                    this.tryEntries.forEach(resetTryEntry);

                    if (!skipTempReset) {
                        for (var name in this) {
                            // Not sure about the optimal order of these conditions:
                            if (name.charAt(0) === "t" &&
                                hasOwn.call(this, name) &&
                                !isNaN(+name.slice(1))) {
                                this[name] = undefined;
                            }
                        }
                    }
                },

                stop: function() {
                    this.done = true;

                    var rootEntry = this.tryEntries[0];
                    var rootRecord = rootEntry.completion;
                    if (rootRecord.type === "throw") {
                        throw rootRecord.arg;
                    }

                    return this.rval;
                },

                dispatchException: function(exception) {
                    if (this.done) {
                        throw exception;
                    }

                    var context = this;
                    function handle(loc, caught) {
                        record.type = "throw";
                        record.arg = exception;
                        context.next = loc;

                        if (caught) {
                            // If the dispatched exception was caught by a catch block,
                            // then let that catch block handle the exception normally.
                            context.method = "next";
                            context.arg = undefined;
                        }

                        return !! caught;
                    }

                    for (var i = this.tryEntries.length - 1; i >= 0; --i) {
                        var entry = this.tryEntries[i];
                        var record = entry.completion;

                        if (entry.tryLoc === "root") {
                            // Exception thrown outside of any try block that could handle
                            // it, so set the completion value of the entire function to
                            // throw the exception.
                            return handle("end");
                        }

                        if (entry.tryLoc <= this.prev) {
                            var hasCatch = hasOwn.call(entry, "catchLoc");
                            var hasFinally = hasOwn.call(entry, "finallyLoc");

                            if (hasCatch && hasFinally) {
                                if (this.prev < entry.catchLoc) {
                                    return handle(entry.catchLoc, true);
                                } else if (this.prev < entry.finallyLoc) {
                                    return handle(entry.finallyLoc);
                                }

                            } else if (hasCatch) {
                                if (this.prev < entry.catchLoc) {
                                    return handle(entry.catchLoc, true);
                                }

                            } else if (hasFinally) {
                                if (this.prev < entry.finallyLoc) {
                                    return handle(entry.finallyLoc);
                                }

                            } else {
                                throw new Error("try statement without catch or finally");
                            }
                        }
                    }
                },

                abrupt: function(type, arg) {
                    for (var i = this.tryEntries.length - 1; i >= 0; --i) {
                        var entry = this.tryEntries[i];
                        if (entry.tryLoc <= this.prev &&
                            hasOwn.call(entry, "finallyLoc") &&
                            this.prev < entry.finallyLoc) {
                            var finallyEntry = entry;
                            break;
                        }
                    }

                    if (finallyEntry &&
                        (type === "break" ||
                            type === "continue") &&
                        finallyEntry.tryLoc <= arg &&
                        arg <= finallyEntry.finallyLoc) {
                        // Ignore the finally entry if control is not jumping to a
                        // location outside the try/catch block.
                        finallyEntry = null;
                    }

                    var record = finallyEntry ? finallyEntry.completion : {};
                    record.type = type;
                    record.arg = arg;

                    if (finallyEntry) {
                        this.method = "next";
                        this.next = finallyEntry.finallyLoc;
                        return ContinueSentinel;
                    }

                    return this.complete(record);
                },

                complete: function(record, afterLoc) {
                    if (record.type === "throw") {
                        throw record.arg;
                    }

                    if (record.type === "break" ||
                        record.type === "continue") {
                        this.next = record.arg;
                    } else if (record.type === "return") {
                        this.rval = this.arg = record.arg;
                        this.method = "return";
                        this.next = "end";
                    } else if (record.type === "normal" && afterLoc) {
                        this.next = afterLoc;
                    }

                    return ContinueSentinel;
                },

                finish: function(finallyLoc) {
                    for (var i = this.tryEntries.length - 1; i >= 0; --i) {
                        var entry = this.tryEntries[i];
                        if (entry.finallyLoc === finallyLoc) {
                            this.complete(entry.completion, entry.afterLoc);
                            resetTryEntry(entry);
                            return ContinueSentinel;
                        }
                    }
                },

                "catch": function(tryLoc) {
                    for (var i = this.tryEntries.length - 1; i >= 0; --i) {
                        var entry = this.tryEntries[i];
                        if (entry.tryLoc === tryLoc) {
                            var record = entry.completion;
                            if (record.type === "throw") {
                                var thrown = record.arg;
                                resetTryEntry(entry);
                            }
                            return thrown;
                        }
                    }

                    // The context.catch method must only be called with a location
                    // argument that corresponds to a known catch block.
                    throw new Error("illegal catch attempt");
                },

                delegateYield: function(iterable, resultName, nextLoc) {
                    this.delegate = {
                        iterator: values(iterable),
                        resultName: resultName,
                        nextLoc: nextLoc
                    };

                    if (this.method === "next") {
                        // Deliberately forget the last sent value so that we don't
                        // accidentally pass it on to the delegate.
                        this.arg = undefined;
                    }

                    return ContinueSentinel;
                }
            };
        })(
            // In sloppy mode, unbound `this` refers to the global object, fallback to
            // Function constructor if we're in global strict mode. That is sadly a form
            // of indirect eval which violates Content Security Policy.
            (function() {
                return this || (typeof self === "object" && self);
            })() || Function("return this")()
        );
    });

    var runtime$1 = /*#__PURE__*/Object.freeze({
        default: runtime,
        __moduleExports: runtime
    });

    var require$$0 = ( runtime$1 && runtime ) || runtime$1;

    /**
     * Copyright (c) 2014-present, Facebook, Inc.
     *
     * This source code is licensed under the MIT license found in the
     * LICENSE file in the root directory of this source tree.
     */

        // This method of obtaining a reference to the global object needs to be
        // kept identical to the way it is obtained in runtime.js
    var g = (function() {
            return this || (typeof self === "object" && self);
        })() || Function("return this")();

    // Use `getOwnPropertyNames` because not all browsers support calling
    // `hasOwnProperty` on the global `self` object in a worker. See #183.
    var hadRuntime = g.regeneratorRuntime &&
        Object.getOwnPropertyNames(g).indexOf("regeneratorRuntime") >= 0;

    // Save the old regeneratorRuntime in case it needs to be restored later.
    var oldRuntime = hadRuntime && g.regeneratorRuntime;

    // Force reevalutation of runtime.js.
    g.regeneratorRuntime = undefined;

    var runtimeModule = require$$0;

    if (hadRuntime) {
        // Restore the original runtime.
        g.regeneratorRuntime = oldRuntime;
    } else {
        // Remove the global property added by runtime.js.
        try {
            delete g.regeneratorRuntime;
        } catch(e) {
            g.regeneratorRuntime = undefined;
        }
    }

    var regenerator = runtimeModule;

    var runtime$2 = createCommonjsModule(function (module) {
        /**
         * Copyright (c) 2014-present, Facebook, Inc.
         *
         * This source code is licensed under the MIT license found in the
         * LICENSE file in the root directory of this source tree.
         */

        !(function(global) {

            var Op = Object.prototype;
            var hasOwn = Op.hasOwnProperty;
            var undefined; // More compressible than void 0.
            var $Symbol = typeof Symbol === "function" ? Symbol : {};
            var iteratorSymbol = $Symbol.iterator || "@@iterator";
            var asyncIteratorSymbol = $Symbol.asyncIterator || "@@asyncIterator";
            var toStringTagSymbol = $Symbol.toStringTag || "@@toStringTag";
            var runtime = global.regeneratorRuntime;
            if (runtime) {
                {
                    // If regeneratorRuntime is defined globally and we're in a module,
                    // make the exports object identical to regeneratorRuntime.
                    module.exports = runtime;
                }
                // Don't bother evaluating the rest of this file if the runtime was
                // already defined globally.
                return;
            }

            // Define the runtime globally (as expected by generated code) as either
            // module.exports (if we're in a module) or a new, empty object.
            runtime = global.regeneratorRuntime = module.exports;

            function wrap(innerFn, outerFn, self, tryLocsList) {
                // If outerFn provided and outerFn.prototype is a Generator, then outerFn.prototype instanceof Generator.
                var protoGenerator = outerFn && outerFn.prototype instanceof Generator ? outerFn : Generator;
                var generator = Object.create(protoGenerator.prototype);
                var context = new Context(tryLocsList || []);

                // The ._invoke method unifies the implementations of the .next,
                // .throw, and .return methods.
                generator._invoke = makeInvokeMethod(innerFn, self, context);

                return generator;
            }
            runtime.wrap = wrap;

            // Try/catch helper to minimize deoptimizations. Returns a completion
            // record like context.tryEntries[i].completion. This interface could
            // have been (and was previously) designed to take a closure to be
            // invoked without arguments, but in all the cases we care about we
            // already have an existing method we want to call, so there's no need
            // to create a new function object. We can even get away with assuming
            // the method takes exactly one argument, since that happens to be true
            // in every case, so we don't have to touch the arguments object. The
            // only additional allocation required is the completion record, which
            // has a stable shape and so hopefully should be cheap to allocate.
            function tryCatch(fn, obj, arg) {
                try {
                    return { type: "normal", arg: fn.call(obj, arg) };
                } catch (err) {
                    return { type: "throw", arg: err };
                }
            }

            var GenStateSuspendedStart = "suspendedStart";
            var GenStateSuspendedYield = "suspendedYield";
            var GenStateExecuting = "executing";
            var GenStateCompleted = "completed";

            // Returning this object from the innerFn has the same effect as
            // breaking out of the dispatch switch statement.
            var ContinueSentinel = {};

            // Dummy constructor functions that we use as the .constructor and
            // .constructor.prototype properties for functions that return Generator
            // objects. For full spec compliance, you may wish to configure your
            // minifier not to mangle the names of these two functions.
            function Generator() {}
            function GeneratorFunction() {}
            function GeneratorFunctionPrototype() {}

            // This is a polyfill for %IteratorPrototype% for environments that
            // don't natively support it.
            var IteratorPrototype = {};
            IteratorPrototype[iteratorSymbol] = function () {
                return this;
            };

            var getProto = Object.getPrototypeOf;
            var NativeIteratorPrototype = getProto && getProto(getProto(values([])));
            if (NativeIteratorPrototype &&
                NativeIteratorPrototype !== Op &&
                hasOwn.call(NativeIteratorPrototype, iteratorSymbol)) {
                // This environment has a native %IteratorPrototype%; use it instead
                // of the polyfill.
                IteratorPrototype = NativeIteratorPrototype;
            }

            var Gp = GeneratorFunctionPrototype.prototype =
                Generator.prototype = Object.create(IteratorPrototype);
            GeneratorFunction.prototype = Gp.constructor = GeneratorFunctionPrototype;
            GeneratorFunctionPrototype.constructor = GeneratorFunction;
            GeneratorFunctionPrototype[toStringTagSymbol] =
                GeneratorFunction.displayName = "GeneratorFunction";

            // Helper for defining the .next, .throw, and .return methods of the
            // Iterator interface in terms of a single ._invoke method.
            function defineIteratorMethods(prototype) {
                ["next", "throw", "return"].forEach(function(method) {
                    prototype[method] = function(arg) {
                        return this._invoke(method, arg);
                    };
                });
            }

            runtime.isGeneratorFunction = function(genFun) {
                var ctor = typeof genFun === "function" && genFun.constructor;
                return ctor
                    ? ctor === GeneratorFunction ||
                    // For the native GeneratorFunction constructor, the best we can
                    // do is to check its .name property.
                    (ctor.displayName || ctor.name) === "GeneratorFunction"
                    : false;
            };

            runtime.mark = function(genFun) {
                if (Object.setPrototypeOf) {
                    Object.setPrototypeOf(genFun, GeneratorFunctionPrototype);
                } else {
                    genFun.__proto__ = GeneratorFunctionPrototype;
                    if (!(toStringTagSymbol in genFun)) {
                        genFun[toStringTagSymbol] = "GeneratorFunction";
                    }
                }
                genFun.prototype = Object.create(Gp);
                return genFun;
            };

            // Within the body of any async function, `await x` is transformed to
            // `yield regeneratorRuntime.awrap(x)`, so that the runtime can test
            // `hasOwn.call(value, "__await")` to determine if the yielded value is
            // meant to be awaited.
            runtime.awrap = function(arg) {
                return { __await: arg };
            };

            function AsyncIterator(generator) {
                function invoke(method, arg, resolve, reject) {
                    var record = tryCatch(generator[method], generator, arg);
                    if (record.type === "throw") {
                        reject(record.arg);
                    } else {
                        var result = record.arg;
                        var value = result.value;
                        if (value &&
                            typeof value === "object" &&
                            hasOwn.call(value, "__await")) {
                            return Promise.resolve(value.__await).then(function(value) {
                                invoke("next", value, resolve, reject);
                            }, function(err) {
                                invoke("throw", err, resolve, reject);
                            });
                        }

                        return Promise.resolve(value).then(function(unwrapped) {
                            // When a yielded Promise is resolved, its final value becomes
                            // the .value of the Promise<{value,done}> result for the
                            // current iteration. If the Promise is rejected, however, the
                            // result for this iteration will be rejected with the same
                            // reason. Note that rejections of yielded Promises are not
                            // thrown back into the generator function, as is the case
                            // when an awaited Promise is rejected. This difference in
                            // behavior between yield and await is important, because it
                            // allows the consumer to decide what to do with the yielded
                            // rejection (swallow it and continue, manually .throw it back
                            // into the generator, abandon iteration, whatever). With
                            // await, by contrast, there is no opportunity to examine the
                            // rejection reason outside the generator function, so the
                            // only option is to throw it from the await expression, and
                            // let the generator function handle the exception.
                            result.value = unwrapped;
                            resolve(result);
                        }, reject);
                    }
                }

                var previousPromise;

                function enqueue(method, arg) {
                    function callInvokeWithMethodAndArg() {
                        return new Promise(function(resolve, reject) {
                            invoke(method, arg, resolve, reject);
                        });
                    }

                    return previousPromise =
                        // If enqueue has been called before, then we want to wait until
                        // all previous Promises have been resolved before calling invoke,
                        // so that results are always delivered in the correct order. If
                        // enqueue has not been called before, then it is important to
                        // call invoke immediately, without waiting on a callback to fire,
                        // so that the async generator function has the opportunity to do
                        // any necessary setup in a predictable way. This predictability
                        // is why the Promise constructor synchronously invokes its
                        // executor callback, and why async functions synchronously
                        // execute code before the first await. Since we implement simple
                        // async functions in terms of async generators, it is especially
                        // important to get this right, even though it requires care.
                        previousPromise ? previousPromise.then(
                            callInvokeWithMethodAndArg,
                            // Avoid propagating failures to Promises returned by later
                            // invocations of the iterator.
                            callInvokeWithMethodAndArg
                        ) : callInvokeWithMethodAndArg();
                }

                // Define the unified helper method that is used to implement .next,
                // .throw, and .return (see defineIteratorMethods).
                this._invoke = enqueue;
            }

            defineIteratorMethods(AsyncIterator.prototype);
            AsyncIterator.prototype[asyncIteratorSymbol] = function () {
                return this;
            };
            runtime.AsyncIterator = AsyncIterator;

            // Note that simple async functions are implemented on top of
            // AsyncIterator objects; they just return a Promise for the value of
            // the final result produced by the iterator.
            runtime.async = function(innerFn, outerFn, self, tryLocsList) {
                var iter = new AsyncIterator(
                    wrap(innerFn, outerFn, self, tryLocsList)
                );

                return runtime.isGeneratorFunction(outerFn)
                    ? iter // If outerFn is a generator, return the full iterator.
                    : iter.next().then(function(result) {
                        return result.done ? result.value : iter.next();
                    });
            };

            function makeInvokeMethod(innerFn, self, context) {
                var state = GenStateSuspendedStart;

                return function invoke(method, arg) {
                    if (state === GenStateExecuting) {
                        throw new Error("Generator is already running");
                    }

                    if (state === GenStateCompleted) {
                        if (method === "throw") {
                            throw arg;
                        }

                        // Be forgiving, per 25.3.3.3.3 of the spec:
                        // https://people.mozilla.org/~jorendorff/es6-draft.html#sec-generatorresume
                        return doneResult();
                    }

                    context.method = method;
                    context.arg = arg;

                    while (true) {
                        var delegate = context.delegate;
                        if (delegate) {
                            var delegateResult = maybeInvokeDelegate(delegate, context);
                            if (delegateResult) {
                                if (delegateResult === ContinueSentinel) continue;
                                return delegateResult;
                            }
                        }

                        if (context.method === "next") {
                            // Setting context._sent for legacy support of Babel's
                            // function.sent implementation.
                            context.sent = context._sent = context.arg;

                        } else if (context.method === "throw") {
                            if (state === GenStateSuspendedStart) {
                                state = GenStateCompleted;
                                throw context.arg;
                            }

                            context.dispatchException(context.arg);

                        } else if (context.method === "return") {
                            context.abrupt("return", context.arg);
                        }

                        state = GenStateExecuting;

                        var record = tryCatch(innerFn, self, context);
                        if (record.type === "normal") {
                            // If an exception is thrown from innerFn, we leave state ===
                            // GenStateExecuting and loop back for another invocation.
                            state = context.done
                                ? GenStateCompleted
                                : GenStateSuspendedYield;

                            if (record.arg === ContinueSentinel) {
                                continue;
                            }

                            return {
                                value: record.arg,
                                done: context.done
                            };

                        } else if (record.type === "throw") {
                            state = GenStateCompleted;
                            // Dispatch the exception by looping back around to the
                            // context.dispatchException(context.arg) call above.
                            context.method = "throw";
                            context.arg = record.arg;
                        }
                    }
                };
            }

            // Call delegate.iterator[context.method](context.arg) and handle the
            // result, either by returning a { value, done } result from the
            // delegate iterator, or by modifying context.method and context.arg,
            // setting context.delegate to null, and returning the ContinueSentinel.
            function maybeInvokeDelegate(delegate, context) {
                var method = delegate.iterator[context.method];
                if (method === undefined) {
                    // A .throw or .return when the delegate iterator has no .throw
                    // method always terminates the yield* loop.
                    context.delegate = null;

                    if (context.method === "throw") {
                        if (delegate.iterator.return) {
                            // If the delegate iterator has a return method, give it a
                            // chance to clean up.
                            context.method = "return";
                            context.arg = undefined;
                            maybeInvokeDelegate(delegate, context);

                            if (context.method === "throw") {
                                // If maybeInvokeDelegate(context) changed context.method from
                                // "return" to "throw", let that override the TypeError below.
                                return ContinueSentinel;
                            }
                        }

                        context.method = "throw";
                        context.arg = new TypeError(
                            "The iterator does not provide a 'throw' method");
                    }

                    return ContinueSentinel;
                }

                var record = tryCatch(method, delegate.iterator, context.arg);

                if (record.type === "throw") {
                    context.method = "throw";
                    context.arg = record.arg;
                    context.delegate = null;
                    return ContinueSentinel;
                }

                var info = record.arg;

                if (! info) {
                    context.method = "throw";
                    context.arg = new TypeError("iterator result is not an object");
                    context.delegate = null;
                    return ContinueSentinel;
                }

                if (info.done) {
                    // Assign the result of the finished delegate to the temporary
                    // variable specified by delegate.resultName (see delegateYield).
                    context[delegate.resultName] = info.value;

                    // Resume execution at the desired location (see delegateYield).
                    context.next = delegate.nextLoc;

                    // If context.method was "throw" but the delegate handled the
                    // exception, let the outer generator proceed normally. If
                    // context.method was "next", forget context.arg since it has been
                    // "consumed" by the delegate iterator. If context.method was
                    // "return", allow the original .return call to continue in the
                    // outer generator.
                    if (context.method !== "return") {
                        context.method = "next";
                        context.arg = undefined;
                    }

                } else {
                    // Re-yield the result returned by the delegate method.
                    return info;
                }

                // The delegate iterator is finished, so forget it and continue with
                // the outer generator.
                context.delegate = null;
                return ContinueSentinel;
            }

            // Define Generator.prototype.{next,throw,return} in terms of the
            // unified ._invoke helper method.
            defineIteratorMethods(Gp);

            Gp[toStringTagSymbol] = "Generator";

            // A Generator should always return itself as the iterator object when the
            // @@iterator function is called on it. Some browsers' implementations of the
            // iterator prototype chain incorrectly implement this, causing the Generator
            // object to not be returned from this call. This ensures that doesn't happen.
            // See https://github.com/facebook/regenerator/issues/274 for more details.
            Gp[iteratorSymbol] = function() {
                return this;
            };

            Gp.toString = function() {
                return "[object Generator]";
            };

            function pushTryEntry(locs) {
                var entry = { tryLoc: locs[0] };

                if (1 in locs) {
                    entry.catchLoc = locs[1];
                }

                if (2 in locs) {
                    entry.finallyLoc = locs[2];
                    entry.afterLoc = locs[3];
                }

                this.tryEntries.push(entry);
            }

            function resetTryEntry(entry) {
                var record = entry.completion || {};
                record.type = "normal";
                delete record.arg;
                entry.completion = record;
            }

            function Context(tryLocsList) {
                // The root entry object (effectively a try statement without a catch
                // or a finally block) gives us a place to store values thrown from
                // locations where there is no enclosing try statement.
                this.tryEntries = [{ tryLoc: "root" }];
                tryLocsList.forEach(pushTryEntry, this);
                this.reset(true);
            }

            runtime.keys = function(object) {
                var keys = [];
                for (var key in object) {
                    keys.push(key);
                }
                keys.reverse();

                // Rather than returning an object with a next method, we keep
                // things simple and return the next function itself.
                return function next() {
                    while (keys.length) {
                        var key = keys.pop();
                        if (key in object) {
                            next.value = key;
                            next.done = false;
                            return next;
                        }
                    }

                    // To avoid creating an additional object, we just hang the .value
                    // and .done properties off the next function object itself. This
                    // also ensures that the minifier will not anonymize the function.
                    next.done = true;
                    return next;
                };
            };

            function values(iterable) {
                if (iterable) {
                    var iteratorMethod = iterable[iteratorSymbol];
                    if (iteratorMethod) {
                        return iteratorMethod.call(iterable);
                    }

                    if (typeof iterable.next === "function") {
                        return iterable;
                    }

                    if (!isNaN(iterable.length)) {
                        var i = -1, next = function next() {
                            while (++i < iterable.length) {
                                if (hasOwn.call(iterable, i)) {
                                    next.value = iterable[i];
                                    next.done = false;
                                    return next;
                                }
                            }

                            next.value = undefined;
                            next.done = true;

                            return next;
                        };

                        return next.next = next;
                    }
                }

                // Return an iterator with no values.
                return { next: doneResult };
            }
            runtime.values = values;

            function doneResult() {
                return { value: undefined, done: true };
            }

            Context.prototype = {
                constructor: Context,

                reset: function(skipTempReset) {
                    this.prev = 0;
                    this.next = 0;
                    // Resetting context._sent for legacy support of Babel's
                    // function.sent implementation.
                    this.sent = this._sent = undefined;
                    this.done = false;
                    this.delegate = null;

                    this.method = "next";
                    this.arg = undefined;

                    this.tryEntries.forEach(resetTryEntry);

                    if (!skipTempReset) {
                        for (var name in this) {
                            // Not sure about the optimal order of these conditions:
                            if (name.charAt(0) === "t" &&
                                hasOwn.call(this, name) &&
                                !isNaN(+name.slice(1))) {
                                this[name] = undefined;
                            }
                        }
                    }
                },

                stop: function() {
                    this.done = true;

                    var rootEntry = this.tryEntries[0];
                    var rootRecord = rootEntry.completion;
                    if (rootRecord.type === "throw") {
                        throw rootRecord.arg;
                    }

                    return this.rval;
                },

                dispatchException: function(exception) {
                    if (this.done) {
                        throw exception;
                    }

                    var context = this;
                    function handle(loc, caught) {
                        record.type = "throw";
                        record.arg = exception;
                        context.next = loc;

                        if (caught) {
                            // If the dispatched exception was caught by a catch block,
                            // then let that catch block handle the exception normally.
                            context.method = "next";
                            context.arg = undefined;
                        }

                        return !! caught;
                    }

                    for (var i = this.tryEntries.length - 1; i >= 0; --i) {
                        var entry = this.tryEntries[i];
                        var record = entry.completion;

                        if (entry.tryLoc === "root") {
                            // Exception thrown outside of any try block that could handle
                            // it, so set the completion value of the entire function to
                            // throw the exception.
                            return handle("end");
                        }

                        if (entry.tryLoc <= this.prev) {
                            var hasCatch = hasOwn.call(entry, "catchLoc");
                            var hasFinally = hasOwn.call(entry, "finallyLoc");

                            if (hasCatch && hasFinally) {
                                if (this.prev < entry.catchLoc) {
                                    return handle(entry.catchLoc, true);
                                } else if (this.prev < entry.finallyLoc) {
                                    return handle(entry.finallyLoc);
                                }

                            } else if (hasCatch) {
                                if (this.prev < entry.catchLoc) {
                                    return handle(entry.catchLoc, true);
                                }

                            } else if (hasFinally) {
                                if (this.prev < entry.finallyLoc) {
                                    return handle(entry.finallyLoc);
                                }

                            } else {
                                throw new Error("try statement without catch or finally");
                            }
                        }
                    }
                },

                abrupt: function(type, arg) {
                    for (var i = this.tryEntries.length - 1; i >= 0; --i) {
                        var entry = this.tryEntries[i];
                        if (entry.tryLoc <= this.prev &&
                            hasOwn.call(entry, "finallyLoc") &&
                            this.prev < entry.finallyLoc) {
                            var finallyEntry = entry;
                            break;
                        }
                    }

                    if (finallyEntry &&
                        (type === "break" ||
                            type === "continue") &&
                        finallyEntry.tryLoc <= arg &&
                        arg <= finallyEntry.finallyLoc) {
                        // Ignore the finally entry if control is not jumping to a
                        // location outside the try/catch block.
                        finallyEntry = null;
                    }

                    var record = finallyEntry ? finallyEntry.completion : {};
                    record.type = type;
                    record.arg = arg;

                    if (finallyEntry) {
                        this.method = "next";
                        this.next = finallyEntry.finallyLoc;
                        return ContinueSentinel;
                    }

                    return this.complete(record);
                },

                complete: function(record, afterLoc) {
                    if (record.type === "throw") {
                        throw record.arg;
                    }

                    if (record.type === "break" ||
                        record.type === "continue") {
                        this.next = record.arg;
                    } else if (record.type === "return") {
                        this.rval = this.arg = record.arg;
                        this.method = "return";
                        this.next = "end";
                    } else if (record.type === "normal" && afterLoc) {
                        this.next = afterLoc;
                    }

                    return ContinueSentinel;
                },

                finish: function(finallyLoc) {
                    for (var i = this.tryEntries.length - 1; i >= 0; --i) {
                        var entry = this.tryEntries[i];
                        if (entry.finallyLoc === finallyLoc) {
                            this.complete(entry.completion, entry.afterLoc);
                            resetTryEntry(entry);
                            return ContinueSentinel;
                        }
                    }
                },

                "catch": function(tryLoc) {
                    for (var i = this.tryEntries.length - 1; i >= 0; --i) {
                        var entry = this.tryEntries[i];
                        if (entry.tryLoc === tryLoc) {
                            var record = entry.completion;
                            if (record.type === "throw") {
                                var thrown = record.arg;
                                resetTryEntry(entry);
                            }
                            return thrown;
                        }
                    }

                    // The context.catch method must only be called with a location
                    // argument that corresponds to a known catch block.
                    throw new Error("illegal catch attempt");
                },

                delegateYield: function(iterable, resultName, nextLoc) {
                    this.delegate = {
                        iterator: values(iterable),
                        resultName: resultName,
                        nextLoc: nextLoc
                    };

                    if (this.method === "next") {
                        // Deliberately forget the last sent value so that we don't
                        // accidentally pass it on to the delegate.
                        this.arg = undefined;
                    }

                    return ContinueSentinel;
                }
            };
        })(
            // In sloppy mode, unbound `this` refers to the global object, fallback to
            // Function constructor if we're in global strict mode. That is sadly a form
            // of indirect eval which violates Content Security Policy.
            (function() { return this })() || Function("return this")()
        );
    });

    function asyncGeneratorStep(gen, resolve, reject, _next, _throw, key, arg) {
        try {
            var info = gen[key](arg);
            var value = info.value;
        } catch (error) {
            reject(error);
            return;
        }

        if (info.done) {
            resolve(value);
        } else {
            Promise.resolve(value).then(_next, _throw);
        }
    }

    function _asyncToGenerator(fn) {
        return function () {
            var self = this,
                args = arguments;
            return new Promise(function (resolve, reject) {
                var gen = fn.apply(self, args);

                function _next(value) {
                    asyncGeneratorStep(gen, resolve, reject, _next, _throw, "next", value);
                }

                function _throw(err) {
                    asyncGeneratorStep(gen, resolve, reject, _next, _throw, "throw", err);
                }

                _next(undefined);
            });
        };
    }

    var asyncToGenerator = _asyncToGenerator;

    function _defineProperty(obj, key, value) {
        if (key in obj) {
            Object.defineProperty(obj, key, {
                value: value,
                enumerable: true,
                configurable: true,
                writable: true
            });
        } else {
            obj[key] = value;
        }

        return obj;
    }

    var defineProperty = _defineProperty;

    function _objectSpread(target) {
        for (var i = 1; i < arguments.length; i++) {
            var source = arguments[i] != null ? arguments[i] : {};
            var ownKeys = Object.keys(source);

            if (typeof Object.getOwnPropertySymbols === 'function') {
                ownKeys = ownKeys.concat(Object.getOwnPropertySymbols(source).filter(function (sym) {
                    return Object.getOwnPropertyDescriptor(source, sym).enumerable;
                }));
            }

            ownKeys.forEach(function (key) {
                defineProperty(target, key, source[key]);
            });
        }

        return target;
    }

    var objectSpread = _objectSpread;

    var bind = function bind(fn, thisArg) {
        return function wrap() {
            var args = new Array(arguments.length);
            for (var i = 0; i < args.length; i++) {
                args[i] = arguments[i];
            }
            return fn.apply(thisArg, args);
        };
    };

    /*!
	 * Determine if an object is a Buffer
	 *
	 * @author   Feross Aboukhadijeh <https://feross.org>
	 * @license  MIT
	 */

    // The _isBuffer check is for Safari 5-7 support, because it's missing
    // Object.prototype.constructor. Remove this eventually
    var isBuffer_1 = function (obj) {
        return obj != null && (isBuffer(obj) || isSlowBuffer(obj) || !!obj._isBuffer)
    };

    function isBuffer (obj) {
        return !!obj.constructor && typeof obj.constructor.isBuffer === 'function' && obj.constructor.isBuffer(obj)
    }

    // For Node v0.10 support. Remove this eventually.
    function isSlowBuffer (obj) {
        return typeof obj.readFloatLE === 'function' && typeof obj.slice === 'function' && isBuffer(obj.slice(0, 0))
    }

    var isBuffer$1 = /*#__PURE__*/Object.freeze({
        default: isBuffer_1,
        __moduleExports: isBuffer_1
    });

    var isBuffer$2 = ( isBuffer$1 && isBuffer_1 ) || isBuffer$1;

    /*global toString:true*/

    // utils is a library of generic helper functions non-specific to axios

    var toString$1 = Object.prototype.toString;

    /**
     * Determine if a value is an Array
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is an Array, otherwise false
     */
    function isArray(val) {
        return toString$1.call(val) === '[object Array]';
    }

    /**
     * Determine if a value is an ArrayBuffer
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is an ArrayBuffer, otherwise false
     */
    function isArrayBuffer(val) {
        return toString$1.call(val) === '[object ArrayBuffer]';
    }

    /**
     * Determine if a value is a FormData
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is an FormData, otherwise false
     */
    function isFormData(val) {
        return (typeof FormData !== 'undefined') && (val instanceof FormData);
    }

    /**
     * Determine if a value is a view on an ArrayBuffer
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a view on an ArrayBuffer, otherwise false
     */
    function isArrayBufferView(val) {
        var result;
        if ((typeof ArrayBuffer !== 'undefined') && (ArrayBuffer.isView)) {
            result = ArrayBuffer.isView(val);
        } else {
            result = (val) && (val.buffer) && (val.buffer instanceof ArrayBuffer);
        }
        return result;
    }

    /**
     * Determine if a value is a String
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a String, otherwise false
     */
    function isString(val) {
        return typeof val === 'string';
    }

    /**
     * Determine if a value is a Number
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a Number, otherwise false
     */
    function isNumber(val) {
        return typeof val === 'number';
    }

    /**
     * Determine if a value is undefined
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if the value is undefined, otherwise false
     */
    function isUndefined(val) {
        return typeof val === 'undefined';
    }

    /**
     * Determine if a value is an Object
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is an Object, otherwise false
     */
    function isObject(val) {
        return val !== null && typeof val === 'object';
    }

    /**
     * Determine if a value is a Date
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a Date, otherwise false
     */
    function isDate(val) {
        return toString$1.call(val) === '[object Date]';
    }

    /**
     * Determine if a value is a File
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a File, otherwise false
     */
    function isFile(val) {
        return toString$1.call(val) === '[object File]';
    }

    /**
     * Determine if a value is a Blob
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a Blob, otherwise false
     */
    function isBlob(val) {
        return toString$1.call(val) === '[object Blob]';
    }

    /**
     * Determine if a value is a Function
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a Function, otherwise false
     */
    function isFunction(val) {
        return toString$1.call(val) === '[object Function]';
    }

    /**
     * Determine if a value is a Stream
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a Stream, otherwise false
     */
    function isStream(val) {
        return isObject(val) && isFunction(val.pipe);
    }

    /**
     * Determine if a value is a URLSearchParams object
     *
     * @param {Object} val The value to test
     * @returns {boolean} True if value is a URLSearchParams object, otherwise false
     */
    function isURLSearchParams(val) {
        return typeof URLSearchParams !== 'undefined' && val instanceof URLSearchParams;
    }

    /**
     * Trim excess whitespace off the beginning and end of a string
     *
     * @param {String} str The String to trim
     * @returns {String} The String freed of excess whitespace
     */
    function trim(str) {
        return str.replace(/^\s*/, '').replace(/\s*$/, '');
    }

    /**
     * Determine if we're running in a standard browser environment
     *
     * This allows axios to run in a web worker, and react-native.
     * Both environments support XMLHttpRequest, but not fully standard globals.
     *
     * web workers:
     *  typeof window -> undefined
     *  typeof document -> undefined
     *
     * react-native:
     *  navigator.product -> 'ReactNative'
     */
    function isStandardBrowserEnv() {
        if (typeof navigator !== 'undefined' && navigator.product === 'ReactNative') {
            return false;
        }
        return (
            typeof window !== 'undefined' &&
            typeof document !== 'undefined'
        );
    }

    /**
     * Iterate over an Array or an Object invoking a function for each item.
     *
     * If `obj` is an Array callback will be called passing
     * the value, index, and complete array for each item.
     *
     * If 'obj' is an Object callback will be called passing
     * the value, key, and complete object for each property.
     *
     * @param {Object|Array} obj The object to iterate
     * @param {Function} fn The callback to invoke for each item
     */
    function forEach(obj, fn) {
        // Don't bother if no value provided
        if (obj === null || typeof obj === 'undefined') {
            return;
        }

        // Force an array if not already something iterable
        if (typeof obj !== 'object') {
            /*eslint no-param-reassign:0*/
            obj = [obj];
        }

        if (isArray(obj)) {
            // Iterate over array values
            for (var i = 0, l = obj.length; i < l; i++) {
                fn.call(null, obj[i], i, obj);
            }
        } else {
            // Iterate over object keys
            for (var key in obj) {
                if (Object.prototype.hasOwnProperty.call(obj, key)) {
                    fn.call(null, obj[key], key, obj);
                }
            }
        }
    }

    /**
     * Accepts varargs expecting each argument to be an object, then
     * immutably merges the properties of each object and returns result.
     *
     * When multiple objects contain the same key the later object in
     * the arguments list will take precedence.
     *
     * Example:
     *
     * ```js
     * var result = merge({foo: 123}, {foo: 456});
     * console.log(result.foo); // outputs 456
     * ```
     *
     * @param {Object} obj1 Object to merge
     * @returns {Object} Result of all merge properties
     */
    function merge(/* obj1, obj2, obj3, ... */) {
        var result = {};
        function assignValue(val, key) {
            if (typeof result[key] === 'object' && typeof val === 'object') {
                result[key] = merge(result[key], val);
            } else {
                result[key] = val;
            }
        }

        for (var i = 0, l = arguments.length; i < l; i++) {
            forEach(arguments[i], assignValue);
        }
        return result;
    }

    /**
     * Extends object a by mutably adding to it the properties of object b.
     *
     * @param {Object} a The object to be extended
     * @param {Object} b The object to copy properties from
     * @param {Object} thisArg The object to bind function to
     * @return {Object} The resulting value of object a
     */
    function extend(a, b, thisArg) {
        forEach(b, function assignValue(val, key) {
            if (thisArg && typeof val === 'function') {
                a[key] = bind(val, thisArg);
            } else {
                a[key] = val;
            }
        });
        return a;
    }

    var utils = {
        isArray: isArray,
        isArrayBuffer: isArrayBuffer,
        isBuffer: isBuffer$2,
        isFormData: isFormData,
        isArrayBufferView: isArrayBufferView,
        isString: isString,
        isNumber: isNumber,
        isObject: isObject,
        isUndefined: isUndefined,
        isDate: isDate,
        isFile: isFile,
        isBlob: isBlob,
        isFunction: isFunction,
        isStream: isStream,
        isURLSearchParams: isURLSearchParams,
        isStandardBrowserEnv: isStandardBrowserEnv,
        forEach: forEach,
        merge: merge,
        extend: extend,
        trim: trim
    };

    var normalizeHeaderName = function normalizeHeaderName(headers, normalizedName) {
        utils.forEach(headers, function processHeader(value, name) {
            if (name !== normalizedName && name.toUpperCase() === normalizedName.toUpperCase()) {
                headers[normalizedName] = value;
                delete headers[name];
            }
        });
    };

    var normalizeHeaderName$1 = /*#__PURE__*/Object.freeze({
        default: normalizeHeaderName,
        __moduleExports: normalizeHeaderName
    });

    /**
     * Update an Error with the specified config, error code, and response.
     *
     * @param {Error} error The error to update.
     * @param {Object} config The config.
     * @param {string} [code] The error code (for example, 'ECONNABORTED').
     * @param {Object} [request] The request.
     * @param {Object} [response] The response.
     * @returns {Error} The error.
     */
    var enhanceError = function enhanceError(error, config, code, request, response) {
        error.config = config;
        if (code) {
            error.code = code;
        }
        error.request = request;
        error.response = response;
        return error;
    };

    var enhanceError$1 = /*#__PURE__*/Object.freeze({
        default: enhanceError,
        __moduleExports: enhanceError
    });

    var enhanceError$2 = ( enhanceError$1 && enhanceError ) || enhanceError$1;

    /**
     * Create an Error with the specified message, config, error code, request and response.
     *
     * @param {string} message The error message.
     * @param {Object} config The config.
     * @param {string} [code] The error code (for example, 'ECONNABORTED').
     * @param {Object} [request] The request.
     * @param {Object} [response] The response.
     * @returns {Error} The created error.
     */
    var createError = function createError(message, config, code, request, response) {
        var error = new Error(message);
        return enhanceError$2(error, config, code, request, response);
    };

    /**
     * Resolve or reject a Promise based on response status.
     *
     * @param {Function} resolve A function that resolves the promise.
     * @param {Function} reject A function that rejects the promise.
     * @param {object} response The response.
     */
    var settle = function settle(resolve, reject, response) {
        var validateStatus = response.config.validateStatus;
        // Note: status is not exposed by XDomainRequest
        if (!response.status || !validateStatus || validateStatus(response.status)) {
            resolve(response);
        } else {
            reject(createError(
                'Request failed with status code ' + response.status,
                response.config,
                null,
                response.request,
                response
            ));
        }
    };

    function encode(val) {
        return encodeURIComponent(val).
        replace(/%40/gi, '@').
        replace(/%3A/gi, ':').
        replace(/%24/g, '$').
        replace(/%2C/gi, ',').
        replace(/%20/g, '+').
        replace(/%5B/gi, '[').
        replace(/%5D/gi, ']');
    }

    /**
     * Build a URL by appending params to the end
     *
     * @param {string} url The base of the url (e.g., http://www.google.com)
     * @param {object} [params] The params to be appended
     * @returns {string} The formatted url
     */
    var buildURL = function buildURL(url, params, paramsSerializer) {
        /*eslint no-param-reassign:0*/
        if (!params) {
            return url;
        }

        var serializedParams;
        if (paramsSerializer) {
            serializedParams = paramsSerializer(params);
        } else if (utils.isURLSearchParams(params)) {
            serializedParams = params.toString();
        } else {
            var parts = [];

            utils.forEach(params, function serialize(val, key) {
                if (val === null || typeof val === 'undefined') {
                    return;
                }

                if (utils.isArray(val)) {
                    key = key + '[]';
                } else {
                    val = [val];
                }

                utils.forEach(val, function parseValue(v) {
                    if (utils.isDate(v)) {
                        v = v.toISOString();
                    } else if (utils.isObject(v)) {
                        v = JSON.stringify(v);
                    }
                    parts.push(encode(key) + '=' + encode(v));
                });
            });

            serializedParams = parts.join('&');
        }

        if (serializedParams) {
            url += (url.indexOf('?') === -1 ? '?' : '&') + serializedParams;
        }

        return url;
    };

    // Headers whose duplicates are ignored by node
    // c.f. https://nodejs.org/api/http.html#http_message_headers
    var ignoreDuplicateOf = [
        'age', 'authorization', 'content-length', 'content-type', 'etag',
        'expires', 'from', 'host', 'if-modified-since', 'if-unmodified-since',
        'last-modified', 'location', 'max-forwards', 'proxy-authorization',
        'referer', 'retry-after', 'user-agent'
    ];

    /**
     * Parse headers into an object
     *
     * ```
     * Date: Wed, 27 Aug 2014 08:58:49 GMT
     * Content-Type: application/json
     * Connection: keep-alive
     * Transfer-Encoding: chunked
     * ```
     *
     * @param {String} headers Headers needing to be parsed
     * @returns {Object} Headers parsed into an object
     */
    var parseHeaders = function parseHeaders(headers) {
        var parsed = {};
        var key;
        var val;
        var i;

        if (!headers) { return parsed; }

        utils.forEach(headers.split('\n'), function parser(line) {
            i = line.indexOf(':');
            key = utils.trim(line.substr(0, i)).toLowerCase();
            val = utils.trim(line.substr(i + 1));

            if (key) {
                if (parsed[key] && ignoreDuplicateOf.indexOf(key) >= 0) {
                    return;
                }
                if (key === 'set-cookie') {
                    parsed[key] = (parsed[key] ? parsed[key] : []).concat([val]);
                } else {
                    parsed[key] = parsed[key] ? parsed[key] + ', ' + val : val;
                }
            }
        });

        return parsed;
    };

    var isURLSameOrigin = (
        utils.isStandardBrowserEnv() ?

            // Standard browser envs have full support of the APIs needed to test
            // whether the request URL is of the same origin as current location.
            (function standardBrowserEnv() {
                var msie = /(msie|trident)/i.test(navigator.userAgent);
                var urlParsingNode = document.createElement('a');
                var originURL;

                /**
                 * Parse a URL to discover it's components
                 *
                 * @param {String} url The URL to be parsed
                 * @returns {Object}
                 */
                function resolveURL(url) {
                    var href = url;

                    if (msie) {
                        // IE needs attribute set twice to normalize properties
                        urlParsingNode.setAttribute('href', href);
                        href = urlParsingNode.href;
                    }

                    urlParsingNode.setAttribute('href', href);

                    // urlParsingNode provides the UrlUtils interface - http://url.spec.whatwg.org/#urlutils
                    return {
                        href: urlParsingNode.href,
                        protocol: urlParsingNode.protocol ? urlParsingNode.protocol.replace(/:$/, '') : '',
                        host: urlParsingNode.host,
                        search: urlParsingNode.search ? urlParsingNode.search.replace(/^\?/, '') : '',
                        hash: urlParsingNode.hash ? urlParsingNode.hash.replace(/^#/, '') : '',
                        hostname: urlParsingNode.hostname,
                        port: urlParsingNode.port,
                        pathname: (urlParsingNode.pathname.charAt(0) === '/') ?
                            urlParsingNode.pathname :
                            '/' + urlParsingNode.pathname
                    };
                }

                originURL = resolveURL(window.location.href);

                /**
                 * Determine if a URL shares the same origin as the current location
                 *
                 * @param {String} requestURL The URL to test
                 * @returns {boolean} True if URL shares the same origin, otherwise false
                 */
                return function isURLSameOrigin(requestURL) {
                    var parsed = (utils.isString(requestURL)) ? resolveURL(requestURL) : requestURL;
                    return (parsed.protocol === originURL.protocol &&
                        parsed.host === originURL.host);
                };
            })() :

            // Non standard browser envs (web workers, react-native) lack needed support.
            (function nonStandardBrowserEnv() {
                return function isURLSameOrigin() {
                    return true;
                };
            })()
    );

    // btoa polyfill for IE<10 courtesy https://github.com/davidchambers/Base64.js

    var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';

    function E() {
        this.message = 'String contains an invalid character';
    }
    E.prototype = new Error;
    E.prototype.code = 5;
    E.prototype.name = 'InvalidCharacterError';

    function btoa(input) {
        var str = String(input);
        var output = '';
        for (
            // initialize result and counter
            var block, charCode, idx = 0, map = chars;
            // if the next str index does not exist:
            //   change the mapping table to "="
            //   check if d has no fractional digits
            str.charAt(idx | 0) || (map = '=', idx % 1);
            // "8 - idx % 1 * 8" generates the sequence 2, 4, 6, 8
            output += map.charAt(63 & block >> 8 - idx % 1 * 8)
        ) {
            charCode = str.charCodeAt(idx += 3 / 4);
            if (charCode > 0xFF) {
                throw new E();
            }
            block = block << 8 | charCode;
        }
        return output;
    }

    var btoa_1 = btoa;

    var cookies = (
        utils.isStandardBrowserEnv() ?

            // Standard browser envs support document.cookie
            (function standardBrowserEnv() {
                return {
                    write: function write(name, value, expires, path, domain, secure) {
                        var cookie = [];
                        cookie.push(name + '=' + encodeURIComponent(value));

                        if (utils.isNumber(expires)) {
                            cookie.push('expires=' + new Date(expires).toGMTString());
                        }

                        if (utils.isString(path)) {
                            cookie.push('path=' + path);
                        }

                        if (utils.isString(domain)) {
                            cookie.push('domain=' + domain);
                        }

                        if (secure === true) {
                            cookie.push('secure');
                        }

                        document.cookie = cookie.join('; ');
                    },

                    read: function read(name) {
                        var match = document.cookie.match(new RegExp('(^|;\\s*)(' + name + ')=([^;]*)'));
                        return (match ? decodeURIComponent(match[3]) : null);
                    },

                    remove: function remove(name) {
                        this.write(name, '', Date.now() - 86400000);
                    }
                };
            })() :

            // Non standard browser env (web workers, react-native) lack needed support.
            (function nonStandardBrowserEnv() {
                return {
                    write: function write() {},
                    read: function read() { return null; },
                    remove: function remove() {}
                };
            })()
    );

    var btoa$1 = (typeof window !== 'undefined' && window.btoa && window.btoa.bind(window)) || btoa_1;

    var xhr = function xhrAdapter(config) {
        return new Promise(function dispatchXhrRequest(resolve, reject) {
            var requestData = config.data;
            var requestHeaders = config.headers;

            if (utils.isFormData(requestData)) {
                delete requestHeaders['Content-Type']; // Let the browser set it
            }

            var request = new XMLHttpRequest();
            var loadEvent = 'onreadystatechange';
            var xDomain = false;

            // For IE 8/9 CORS support
            // Only supports POST and GET calls and doesn't returns the response headers.
            // DON'T do this for testing b/c XMLHttpRequest is mocked, not XDomainRequest.
            if (typeof window !== 'undefined' &&
                window.XDomainRequest && !('withCredentials' in request) &&
                !isURLSameOrigin(config.url)) {
                request = new window.XDomainRequest();
                loadEvent = 'onload';
                xDomain = true;
                request.onprogress = function handleProgress() {};
                request.ontimeout = function handleTimeout() {};
            }

            // HTTP basic authentication
            if (config.auth) {
                var username = config.auth.username || '';
                var password = config.auth.password || '';
                requestHeaders.Authorization = 'Basic ' + btoa$1(username + ':' + password);
            }

            request.open(config.method.toUpperCase(), buildURL(config.url, config.params, config.paramsSerializer), true);

            // Set the request timeout in MS
            request.timeout = config.timeout;

            // Listen for ready state
            request[loadEvent] = function handleLoad() {
                if (!request || (request.readyState !== 4 && !xDomain)) {
                    return;
                }

                // The request errored out and we didn't get a response, this will be
                // handled by onerror instead
                // With one exception: request that using file: protocol, most browsers
                // will return status as 0 even though it's a successful request
                if (request.status === 0 && !(request.responseURL && request.responseURL.indexOf('file:') === 0)) {
                    return;
                }

                // Prepare the response
                var responseHeaders = 'getAllResponseHeaders' in request ? parseHeaders(request.getAllResponseHeaders()) : null;
                var responseData = !config.responseType || config.responseType === 'text' ? request.responseText : request.response;
                var response = {
                    data: responseData,
                    // IE sends 1223 instead of 204 (https://github.com/axios/axios/issues/201)
                    status: request.status === 1223 ? 204 : request.status,
                    statusText: request.status === 1223 ? 'No Content' : request.statusText,
                    headers: responseHeaders,
                    config: config,
                    request: request
                };

                settle(resolve, reject, response);

                // Clean up request
                request = null;
            };

            // Handle low level network errors
            request.onerror = function handleError() {
                // Real errors are hidden from us by the browser
                // onerror should only fire if it's a network error
                reject(createError('Network Error', config, null, request));

                // Clean up request
                request = null;
            };

            // Handle timeout
            request.ontimeout = function handleTimeout() {
                reject(createError('timeout of ' + config.timeout + 'ms exceeded', config, 'ECONNABORTED',
                    request));

                // Clean up request
                request = null;
            };

            // Add xsrf header
            // This is only done if running in a standard browser environment.
            // Specifically not if we're in a web worker, or react-native.
            if (utils.isStandardBrowserEnv()) {
                var cookies$$1 = cookies;

                // Add xsrf header
                var xsrfValue = (config.withCredentials || isURLSameOrigin(config.url)) && config.xsrfCookieName ?
                    cookies$$1.read(config.xsrfCookieName) :
                    undefined;

                if (xsrfValue) {
                    requestHeaders[config.xsrfHeaderName] = xsrfValue;
                }
            }

            // Add headers to the request
            if ('setRequestHeader' in request) {
                utils.forEach(requestHeaders, function setRequestHeader(val, key) {
                    if (typeof requestData === 'undefined' && key.toLowerCase() === 'content-type') {
                        // Remove Content-Type if data is undefined
                        delete requestHeaders[key];
                    } else {
                        // Otherwise add header to the request
                        request.setRequestHeader(key, val);
                    }
                });
            }

            // Add withCredentials to request if needed
            if (config.withCredentials) {
                request.withCredentials = true;
            }

            // Add responseType to request if needed
            if (config.responseType) {
                try {
                    request.responseType = config.responseType;
                } catch (e) {
                    // Expected DOMException thrown by browsers not compatible XMLHttpRequest Level 2.
                    // But, this can be suppressed for 'json' type as it can be parsed by default 'transformResponse' function.
                    if (config.responseType !== 'json') {
                        throw e;
                    }
                }
            }

            // Handle progress if needed
            if (typeof config.onDownloadProgress === 'function') {
                request.addEventListener('progress', config.onDownloadProgress);
            }

            // Not all browsers support upload events
            if (typeof config.onUploadProgress === 'function' && request.upload) {
                request.upload.addEventListener('progress', config.onUploadProgress);
            }

            if (config.cancelToken) {
                // Handle cancellation
                config.cancelToken.promise.then(function onCanceled(cancel) {
                    if (!request) {
                        return;
                    }

                    request.abort();
                    reject(cancel);
                    // Clean up request
                    request = null;
                });
            }

            if (requestData === undefined) {
                requestData = null;
            }

            // Send the request
            request.send(requestData);
        });
    };

    var normalizeHeaderName$2 = ( normalizeHeaderName$1 && normalizeHeaderName ) || normalizeHeaderName$1;

    var DEFAULT_CONTENT_TYPE = {
        'Content-Type': 'application/x-www-form-urlencoded'
    };

    function setContentTypeIfUnset(headers, value) {
        if (!utils.isUndefined(headers) && utils.isUndefined(headers['Content-Type'])) {
            headers['Content-Type'] = value;
        }
    }

    function getDefaultAdapter() {
        var adapter;
        if (typeof XMLHttpRequest !== 'undefined') {
            // For browsers use XHR adapter
            adapter = xhr;
        } else if (typeof process !== 'undefined') {
            // For node use HTTP adapter
            adapter = xhr;
        }
        return adapter;
    }

    var defaults = {
        adapter: getDefaultAdapter(),

        transformRequest: [function transformRequest(data, headers) {
            normalizeHeaderName$2(headers, 'Content-Type');
            if (utils.isFormData(data) ||
                utils.isArrayBuffer(data) ||
                utils.isBuffer(data) ||
                utils.isStream(data) ||
                utils.isFile(data) ||
                utils.isBlob(data)
            ) {
                return data;
            }
            if (utils.isArrayBufferView(data)) {
                return data.buffer;
            }
            if (utils.isURLSearchParams(data)) {
                setContentTypeIfUnset(headers, 'application/x-www-form-urlencoded;charset=utf-8');
                return data.toString();
            }
            if (utils.isObject(data)) {
                setContentTypeIfUnset(headers, 'application/json;charset=utf-8');
                return JSON.stringify(data);
            }
            return data;
        }],

        transformResponse: [function transformResponse(data) {
            /*eslint no-param-reassign:0*/
            if (typeof data === 'string') {
                try {
                    data = JSON.parse(data);
                } catch (e) { /* Ignore */ }
            }
            return data;
        }],

        /**
         * A timeout in milliseconds to abort a request. If set to 0 (default) a
         * timeout is not created.
         */
        timeout: 0,

        xsrfCookieName: 'XSRF-TOKEN',
        xsrfHeaderName: 'X-XSRF-TOKEN',

        maxContentLength: -1,

        validateStatus: function validateStatus(status) {
            return status >= 200 && status < 300;
        }
    };

    defaults.headers = {
        common: {
            'Accept': 'application/json, text/plain, */*'
        }
    };

    utils.forEach(['delete', 'get', 'head'], function forEachMethodNoData(method) {
        defaults.headers[method] = {};
    });

    utils.forEach(['post', 'put', 'patch'], function forEachMethodWithData(method) {
        defaults.headers[method] = utils.merge(DEFAULT_CONTENT_TYPE);
    });

    var defaults_1 = defaults;

    function InterceptorManager() {
        this.handlers = [];
    }

    /**
     * Add a new interceptor to the stack
     *
     * @param {Function} fulfilled The function to handle `then` for a `Promise`
     * @param {Function} rejected The function to handle `reject` for a `Promise`
     *
     * @return {Number} An ID used to remove interceptor later
     */
    InterceptorManager.prototype.use = function use(fulfilled, rejected) {
        this.handlers.push({
            fulfilled: fulfilled,
            rejected: rejected
        });
        return this.handlers.length - 1;
    };

    /**
     * Remove an interceptor from the stack
     *
     * @param {Number} id The ID that was returned by `use`
     */
    InterceptorManager.prototype.eject = function eject(id) {
        if (this.handlers[id]) {
            this.handlers[id] = null;
        }
    };

    /**
     * Iterate over all the registered interceptors
     *
     * This method is particularly useful for skipping over any
     * interceptors that may have become `null` calling `eject`.
     *
     * @param {Function} fn The function to call for each interceptor
     */
    InterceptorManager.prototype.forEach = function forEach(fn) {
        utils.forEach(this.handlers, function forEachHandler(h) {
            if (h !== null) {
                fn(h);
            }
        });
    };

    var InterceptorManager_1 = InterceptorManager;

    /**
     * Transform the data for a request or a response
     *
     * @param {Object|String} data The data to be transformed
     * @param {Array} headers The headers for the request or response
     * @param {Array|Function} fns A single function or Array of functions
     * @returns {*} The resulting transformed data
     */
    var transformData = function transformData(data, headers, fns) {
        /*eslint no-param-reassign:0*/
        utils.forEach(fns, function transform(fn) {
            data = fn(data, headers);
        });

        return data;
    };

    var isCancel = function isCancel(value) {
        return !!(value && value.__CANCEL__);
    };

    /**
     * Determines whether the specified URL is absolute
     *
     * @param {string} url The URL to test
     * @returns {boolean} True if the specified URL is absolute, otherwise false
     */
    var isAbsoluteURL = function isAbsoluteURL(url) {
        // A URL is considered absolute if it begins with "<scheme>://" or "//" (protocol-relative URL).
        // RFC 3986 defines scheme name as a sequence of characters beginning with a letter and followed
        // by any combination of letters, digits, plus, period, or hyphen.
        return /^([a-z][a-z\d\+\-\.]*:)?\/\//i.test(url);
    };

    /**
     * Creates a new URL by combining the specified URLs
     *
     * @param {string} baseURL The base URL
     * @param {string} relativeURL The relative URL
     * @returns {string} The combined URL
     */
    var combineURLs = function combineURLs(baseURL, relativeURL) {
        return relativeURL
            ? baseURL.replace(/\/+$/, '') + '/' + relativeURL.replace(/^\/+/, '')
            : baseURL;
    };

    /**
     * Throws a `Cancel` if cancellation has been requested.
     */
    function throwIfCancellationRequested(config) {
        if (config.cancelToken) {
            config.cancelToken.throwIfRequested();
        }
    }

    /**
     * Dispatch a request to the server using the configured adapter.
     *
     * @param {object} config The config that is to be used for the request
     * @returns {Promise} The Promise to be fulfilled
     */
    var dispatchRequest = function dispatchRequest(config) {
        throwIfCancellationRequested(config);

        // Support baseURL config
        if (config.baseURL && !isAbsoluteURL(config.url)) {
            config.url = combineURLs(config.baseURL, config.url);
        }

        // Ensure headers exist
        config.headers = config.headers || {};

        // Transform request data
        config.data = transformData(
            config.data,
            config.headers,
            config.transformRequest
        );

        // Flatten headers
        config.headers = utils.merge(
            config.headers.common || {},
            config.headers[config.method] || {},
            config.headers || {}
        );

        utils.forEach(
            ['delete', 'get', 'head', 'post', 'put', 'patch', 'common'],
            function cleanHeaderConfig(method) {
                delete config.headers[method];
            }
        );

        var adapter = config.adapter || defaults_1.adapter;

        return adapter(config).then(function onAdapterResolution(response) {
            throwIfCancellationRequested(config);

            // Transform response data
            response.data = transformData(
                response.data,
                response.headers,
                config.transformResponse
            );

            return response;
        }, function onAdapterRejection(reason) {
            if (!isCancel(reason)) {
                throwIfCancellationRequested(config);

                // Transform response data
                if (reason && reason.response) {
                    reason.response.data = transformData(
                        reason.response.data,
                        reason.response.headers,
                        config.transformResponse
                    );
                }
            }

            return Promise.reject(reason);
        });
    };

    /**
     * Create a new instance of Axios
     *
     * @param {Object} instanceConfig The default config for the instance
     */
    function Axios(instanceConfig) {
        this.defaults = instanceConfig;
        this.interceptors = {
            request: new InterceptorManager_1(),
            response: new InterceptorManager_1()
        };
    }

    /**
     * Dispatch a request
     *
     * @param {Object} config The config specific for this request (merged with this.defaults)
     */
    Axios.prototype.request = function request(config) {
        /*eslint no-param-reassign:0*/
        // Allow for axios('example/url'[, config]) a la fetch API
        if (typeof config === 'string') {
            config = utils.merge({
                url: arguments[0]
            }, arguments[1]);
        }

        config = utils.merge(defaults_1, {method: 'get'}, this.defaults, config);
        config.method = config.method.toLowerCase();

        // Hook up interceptors middleware
        var chain = [dispatchRequest, undefined];
        var promise = Promise.resolve(config);

        this.interceptors.request.forEach(function unshiftRequestInterceptors(interceptor) {
            chain.unshift(interceptor.fulfilled, interceptor.rejected);
        });

        this.interceptors.response.forEach(function pushResponseInterceptors(interceptor) {
            chain.push(interceptor.fulfilled, interceptor.rejected);
        });

        while (chain.length) {
            promise = promise.then(chain.shift(), chain.shift());
        }

        return promise;
    };

    // Provide aliases for supported request methods
    utils.forEach(['delete', 'get', 'head', 'options'], function forEachMethodNoData(method) {
        /*eslint func-names:0*/
        Axios.prototype[method] = function(url, config) {
            return this.request(utils.merge(config || {}, {
                method: method,
                url: url
            }));
        };
    });

    utils.forEach(['post', 'put', 'patch'], function forEachMethodWithData(method) {
        /*eslint func-names:0*/
        Axios.prototype[method] = function(url, data, config) {
            return this.request(utils.merge(config || {}, {
                method: method,
                url: url,
                data: data
            }));
        };
    });

    var Axios_1 = Axios;

    /**
     * A `Cancel` is an object that is thrown when an operation is canceled.
     *
     * @class
     * @param {string=} message The message.
     */
    function Cancel(message) {
        this.message = message;
    }

    Cancel.prototype.toString = function toString() {
        return 'Cancel' + (this.message ? ': ' + this.message : '');
    };

    Cancel.prototype.__CANCEL__ = true;

    var Cancel_1 = Cancel;

    /**
     * A `CancelToken` is an object that can be used to request cancellation of an operation.
     *
     * @class
     * @param {Function} executor The executor function.
     */
    function CancelToken(executor) {
        if (typeof executor !== 'function') {
            throw new TypeError('executor must be a function.');
        }

        var resolvePromise;
        this.promise = new Promise(function promiseExecutor(resolve) {
            resolvePromise = resolve;
        });

        var token = this;
        executor(function cancel(message) {
            if (token.reason) {
                // Cancellation has already been requested
                return;
            }

            token.reason = new Cancel_1(message);
            resolvePromise(token.reason);
        });
    }

    /**
     * Throws a `Cancel` if cancellation has been requested.
     */
    CancelToken.prototype.throwIfRequested = function throwIfRequested() {
        if (this.reason) {
            throw this.reason;
        }
    };

    /**
     * Returns an object that contains a new `CancelToken` and a function that, when called,
     * cancels the `CancelToken`.
     */
    CancelToken.source = function source() {
        var cancel;
        var token = new CancelToken(function executor(c) {
            cancel = c;
        });
        return {
            token: token,
            cancel: cancel
        };
    };

    var CancelToken_1 = CancelToken;

    /**
     * Syntactic sugar for invoking a function and expanding an array for arguments.
     *
     * Common use case would be to use `Function.prototype.apply`.
     *
     *  ```js
     *  function f(x, y, z) {}
     *  var args = [1, 2, 3];
     *  f.apply(null, args);
     *  ```
     *
     * With `spread` this example can be re-written.
     *
     *  ```js
     *  spread(function(x, y, z) {})([1, 2, 3]);
     *  ```
     *
     * @param {Function} callback
     * @returns {Function}
     */
    var spread = function spread(callback) {
        return function wrap(arr) {
            return callback.apply(null, arr);
        };
    };

    /**
     * Create an instance of Axios
     *
     * @param {Object} defaultConfig The default config for the instance
     * @return {Axios} A new instance of Axios
     */
    function createInstance(defaultConfig) {
        var context = new Axios_1(defaultConfig);
        var instance = bind(Axios_1.prototype.request, context);

        // Copy axios.prototype to instance
        utils.extend(instance, Axios_1.prototype, context);

        // Copy context to instance
        utils.extend(instance, context);

        return instance;
    }

    // Create the default instance to be exported
    var axios = createInstance(defaults_1);

    // Expose Axios class to allow class inheritance
    axios.Axios = Axios_1;

    // Factory for creating new instances
    axios.create = function create(instanceConfig) {
        return createInstance(utils.merge(defaults_1, instanceConfig));
    };

    // Expose Cancel & CancelToken
    axios.Cancel = Cancel_1;
    axios.CancelToken = CancelToken_1;
    axios.isCancel = isCancel;

    // Expose all/spread
    axios.all = function all(promises) {
        return Promise.all(promises);
    };
    axios.spread = spread;

    var axios_1 = axios;

    // Allow use of default import syntax in TypeScript
    var default_1 = axios;
    axios_1.default = default_1;

    var axios$1 = axios_1;

    var errors = {
        InvalidNumberOfSolidityArgs: function InvalidNumberOfSolidityArgs() {
            return 'Invalid number of arguments to Solidity function';
        },
        InvalidNumberOfRPCParams: function InvalidNumberOfRPCParams() {
            return 'Invalid number of input parameters to RPC method';
        },
        InvalidRPCMethod: function InvalidRPCMethod(params) {
            return "JSONRPC api name should be specified but only found params: \"".concat(JSON.stringify(params), "\"");
        },
        InvalidConnection: function InvalidConnection(host) {
            return "CONNECTION ERROR: Couldn't connect to node ".concat(host);
        },
        InvalidConn: function InvalidConn() {
            return 'Conn not set or invalid';
        },
        InvalidResponse: function InvalidResponse(result) {
            return !!result && !!result.error && !!result.error.message ? result.error.message : "Invalid JSON RPC response: ".concat(JSON.stringify(result));
        },
        ConnectionTimeout: function ConnectionTimeout(ms) {
            return "CONNECTION TIMEOUT: timeout of ".concat(ms, " ms achived");
        }
    };

    var DEFAULT_HTTP_HOST = 'http://127.0.0.1:8001';
    var POLL_DURATION = 3000;
    var MAX_POLL_RETRY = 5;

    var HttpConn =
        /*#__PURE__*/
        function () {
            function HttpConn(host, timeout, username, password, headers) {
                classCallCheck(this, HttpConn);

                this.host = host || DEFAULT_HTTP_HOST;
                this.timeout = timeout || 0;
                var config = {
                    baseURL: this.host,
                    timeout: this.timeout,
                    headers: {
                        'Content-Type': 'application/json'
                    }
                };

                if (username && password) {
                    config.auth = {
                        username: username,
                        password: password
                    };
                }

                if (headers) {
                    config.headers = objectSpread({}, config.headers, headers);
                }

                this.axiosInstance = axios$1.create(config);
            }

            createClass(HttpConn, [{
                key: "send",
                value: function () {
                    var _send = asyncToGenerator(
                        /*#__PURE__*/
                        regenerator.mark(function _callee(payload) {
                            var response;
                            return regenerator.wrap(function _callee$(_context) {
                                while (1) {
                                    switch (_context.prev = _context.next) {
                                        case 0:
                                            _context.prev = 0;
                                            _context.next = 3;
                                            return this.axiosInstance.post('', payload);

                                        case 3:
                                            response = _context.sent;
                                            _context.next = 10;
                                            break;

                                        case 6:
                                            _context.prev = 6;
                                            _context.t0 = _context["catch"](0);
                                            console.error('send fail:', _context.t0.message);
                                            throw new Error(errors.InvalidConnection(this.host));

                                        case 10:
                                            return _context.abrupt("return", response.data);

                                        case 11:
                                        case "end":
                                            return _context.stop();
                                    }
                                }
                            }, _callee, this, [[0, 6]]);
                        }));

                    return function send(_x) {
                        return _send.apply(this, arguments);
                    };
                }()
            }]);

            return HttpConn;
        }();

    var f$2 = {}.propertyIsEnumerable;

    var _objectPie = {
        f: f$2
    };

    var isEnum = _objectPie.f;
    var _objectToArray = function (isEntries) {
        return function (it) {
            var O = _toIobject(it);
            var keys = _objectKeys(O);
            var length = keys.length;
            var i = 0;
            var result = [];
            var key;
            while (length > i) if (isEnum.call(O, key = keys[i++])) {
                result.push(isEntries ? [key, O[key]] : O[key]);
            } return result;
        };
    };

    // https://github.com/tc39/proposal-object-values-entries

    var $values = _objectToArray(false);

    _export(_export.S, 'Object', {
        values: function values(it) {
            return $values(it);
        }
    });

    var messageId = 0;
    /**
     * Create jsonrpc payload object
     *
     * @param {string} method the method name of jsonrpc call
     * @param {Array?} params an array of method params
     * @returns {object} valid jsonrpc payload object
     */

    function toPayload(method, params) {
        if (!method) {
            throw new Error(errors.InvalidRPCMethod(params));
        } // advance message ID


        messageId++;
        return {
            jsonrpc: '2.0',
            id: messageId,
            method: method,
            params: params || []
        };
    }
    /**
     * Check if jsonrpc response is valid
     *
     * @param {object|Array} response
     * @param {string} response.error
     * @param {string} response.jsonrpc
     * @param {string|number} response.id
     * @param {*} response.result
     * @returns {Boolean} true if response is valid, otherwise false
     */


    function isValidResponse(response) {
        return Array.isArray(response) ? response.every(validateSingleMessage) : validateSingleMessage(response);

        function validateSingleMessage(message) {
            return !!message && !message.error && message.jsonrpc === '2.0' && (typeof message.id === 'number' || typeof message.id === 'string') && message.result !== undefined; // undefined is not valid json object
        }
    }
    /**
     * Create jsonrpc batch payload object
     *
     * @param {Array} messages an array of objects to create jsonrpc payload object method
     * @param {string} messages.method the method name of jsonrpc call
     * @param {Array?} messages.params an array of method params
     * @returns {Array} batch payload
     */


    function toBatchPayload(messages) {
        return messages.map(function (message) {
            return toPayload(message.method, message.params);
        });
    }

    var jsonrpc = {
        toPayload: toPayload,
        isValidResponse: isValidResponse,
        toBatchPayload: toBatchPayload
    };

    /**
     * It's responsible for passing messages to conn
     */

    var Requester =
        /*#__PURE__*/
        function () {
            function Requester(conn) {
                classCallCheck(this, Requester);

                this.conn = conn;
                this.idGenerator = 0; // used for generate watchId

                this.watchers = {}; // key is watchId, value is timer
            }
            /**
             * Send request to lemo node asynchronously over RPC
             *
             * @param {string} method
             * @param {Array?} params an array of method params
             * @return {Promise}
             */


            createClass(Requester, [{
                key: "send",
                value: function () {
                    var _send = asyncToGenerator(
                        /*#__PURE__*/
                        regenerator.mark(function _callee(method, params) {
                            var payload, response;
                            return regenerator.wrap(function _callee$(_context) {
                                while (1) {
                                    switch (_context.prev = _context.next) {
                                        case 0:
                                            if (this.conn) {
                                                _context.next = 2;
                                                break;
                                            }

                                            throw new Error(errors.InvalidConn());

                                        case 2:
                                            payload = jsonrpc.toPayload(method, params);
                                            response = this.conn.send(payload);

                                            if (!(response && typeof response.then === 'function')) {
                                                _context.next = 8;
                                                break;
                                            }

                                            _context.next = 7;
                                            return response;

                                        case 7:
                                            response = _context.sent;

                                        case 8:
                                            if (jsonrpc.isValidResponse(response)) {
                                                _context.next = 10;
                                                break;
                                            }

                                            throw new Error(errors.InvalidResponse(response));

                                        case 10:
                                            return _context.abrupt("return", response.result);

                                        case 11:
                                        case "end":
                                            return _context.stop();
                                    }
                                }
                            }, _callee, this);
                        }));

                    return function send(_x, _x2) {
                        return _send.apply(this, arguments);
                    };
                }()
                /**
                 * Send batch request to lemo node asynchronously over RPC
                 *
                 * @param {object[]} data
                 * @param {string} data.method
                 * @param {Array?} data.params an array of method params
                 * @return {Promise}
                 */

            }, {
                key: "sendBatch",
                value: function () {
                    var _sendBatch = asyncToGenerator(
                        /*#__PURE__*/
                        regenerator.mark(function _callee2(data) {
                            var payload, response;
                            return regenerator.wrap(function _callee2$(_context2) {
                                while (1) {
                                    switch (_context2.prev = _context2.next) {
                                        case 0:
                                            if (this.conn) {
                                                _context2.next = 2;
                                                break;
                                            }

                                            throw new Error(errors.InvalidConn());

                                        case 2:
                                            payload = jsonrpc.toBatchPayload(data);
                                            response = this.conn.send(payload);

                                            if (!(response && typeof response.then === 'function')) {
                                                _context2.next = 8;
                                                break;
                                            }

                                            _context2.next = 7;
                                            return response;

                                        case 7:
                                            response = _context2.sent;

                                        case 8:
                                            if (Array.isArray(response)) {
                                                _context2.next = 10;
                                                break;
                                            }

                                            throw new Error(errors.InvalidResponse(response));

                                        case 10:
                                            response.forEach(function (result) {
                                                if (!jsonrpc.isValidResponse(result)) {
                                                    throw new Error(errors.InvalidResponse(result));
                                                }
                                            });
                                            return _context2.abrupt("return", response);

                                        case 12:
                                        case "end":
                                            return _context2.stop();
                                    }
                                }
                            }, _callee2, this);
                        }));

                    return function sendBatch(_x3) {
                        return _sendBatch.apply(this, arguments);
                    };
                }()
                /**
                 * poll till the response changed
                 *
                 * @param {string} method
                 * @param {Array?} params An array of method params
                 * @param {Function} callback The function to receive result. it must be like function(result, error)
                 * @return {number} The watchId which is used to stop watching
                 */

            }, {
                key: "watch",
                value: function watch(method, params, callback) {
                    var _this = this;

                    if (typeof params === 'function' && typeof callback === 'undefined') {
                        // no params
                        callback = params;
                        params = undefined;
                    }

                    var lastSig;
                    var errCount = 0;
                    var newWatchId = this.idGenerator++;
                    this.watchers[newWatchId] = setInterval(
                        /*#__PURE__*/
                        asyncToGenerator(
                            /*#__PURE__*/
                            regenerator.mark(function _callee3() {
                                var result, error, sig;
                                return regenerator.wrap(function _callee3$(_context3) {
                                    while (1) {
                                        switch (_context3.prev = _context3.next) {
                                            case 0:
                                                _context3.prev = 0;
                                                _context3.next = 3;
                                                return _this.send(method, params);

                                            case 3:
                                                result = _context3.sent;
                                                errCount = 0;
                                                sig = getSig(result);

                                                if (!(sig === lastSig)) {
                                                    _context3.next = 8;
                                                    break;
                                                }

                                                return _context3.abrupt("return");

                                            case 8:
                                                lastSig = sig;
                                                _context3.next = 18;
                                                break;

                                            case 11:
                                                _context3.prev = 11;
                                                _context3.t0 = _context3["catch"](0);
                                                console.error('watch fail:', _context3.t0);

                                                if (!(++errCount <= MAX_POLL_RETRY)) {
                                                    _context3.next = 16;
                                                    break;
                                                }

                                                return _context3.abrupt("return");

                                            case 16:
                                                error = _context3.t0;

                                                _this.stopWatch(newWatchId);

                                            case 18:
                                                // put callback out of try block to expose user's error
                                                callback(result, error);

                                            case 19:
                                            case "end":
                                                return _context3.stop();
                                        }
                                    }
                                }, _callee3, this, [[0, 11]]);
                            })), POLL_DURATION);
                    return newWatchId;
                }
            }, {
                key: "stopWatch",
                value: function stopWatch(watchId) {
                    if (this.watchers[watchId] > 0) {
                        clearInterval(this.watchers[watchId]);
                        delete this.watchers[watchId];
                    }
                }
                /**
                 * stop all watching
                 */

            }, {
                key: "reset",
                value: function reset() {
                    var timers = this.watchers;
                    this.watchers = {};
                    Object.values(timers).forEach(clearInterval);
                }
            }]);

            return Requester;
        }(); // get the signature of data


    function getSig(data) {
        return JSON.stringify(data);
    }

    function _arrayWithHoles(arr) {
        if (Array.isArray(arr)) return arr;
    }

    var arrayWithHoles = _arrayWithHoles;

    function _iterableToArrayLimit(arr, i) {
        var _arr = [];
        var _n = true;
        var _d = false;
        var _e = undefined;

        try {
            for (var _i = arr[Symbol.iterator](), _s; !(_n = (_s = _i.next()).done); _n = true) {
                _arr.push(_s.value);

                if (i && _arr.length === i) break;
            }
        } catch (err) {
            _d = true;
            _e = err;
        } finally {
            try {
                if (!_n && _i["return"] != null) _i["return"]();
            } finally {
                if (_d) throw _e;
            }
        }

        return _arr;
    }

    var iterableToArrayLimit = _iterableToArrayLimit;

    function _nonIterableRest() {
        throw new TypeError("Invalid attempt to destructure non-iterable instance");
    }

    var nonIterableRest = _nonIterableRest;

    function _slicedToArray(arr, i) {
        return arrayWithHoles(arr) || iterableToArrayLimit(arr, i) || nonIterableRest();
    }

    var slicedToArray = _slicedToArray;

    // https://github.com/tc39/proposal-object-values-entries

    var $entries = _objectToArray(true);

    _export(_export.S, 'Object', {
        entries: function entries(it) {
            return $entries(it);
        }
    });

    function isHash(hashOrHeight) {
        return typeof hashOrHeight === 'string' && hashOrHeight.toLowerCase().startsWith('0x');
    }
    function parseBlock(block) {
        return block;
    }
    function strToBigNumber(num) {
        return new BigNumber(num);
    }

    var MODULE_NAME = 'account';
    var apiList = {
        getAccount: {
            method: "".concat(MODULE_NAME, "_getAccount")
        },
        getBalance: {
            method: "".concat(MODULE_NAME, "_getBalance"),
            outputFormatter: strToBigNumber
        }
    };
    var apis = Object.entries(apiList).map(function (_ref) {
        var _ref2 = slicedToArray(_ref, 2),
            key = _ref2[0],
            value = _ref2[1];

        if (typeof value === 'function') {
            return {
                name: key,
                call: value
            };
        }

        return objectSpread({
            name: key
        }, value);
    });
    var account = {
        moduleName: MODULE_NAME,
        apis: apis
    };

    var MODULE_NAME$1 = 'chain';
    var apiList$1 = {
        getCurrentBlock: function getCurrentBlock(requester, stable, withTxList) {
            var apiName = stable ? 'currentBlock' : 'latestStableBlock';
            return requester.send("".concat(MODULE_NAME$1, "_").concat(apiName), [withTxList]).then(parseBlock);
        },
        getBlock: function getBlock(requester, hashOrHeight, withTxList) {
            var apiName = isHash(hashOrHeight) ? 'getBlockByHash' : 'getBlockByHeight';
            return requester.send("".concat(MODULE_NAME$1, "_").concat(apiName), [hashOrHeight, withTxList]).then(parseBlock);
        },
        getCurrentHeight: function getCurrentHeight(requester, stable) {
            var apiName = stable ? 'currentHeight' : 'latestStableHeight';
            return requester.send("".concat(MODULE_NAME$1, "_").concat(apiName));
        },
        getGenesis: {
            method: "".concat(MODULE_NAME$1, "_genesis")
        },
        getChainID: {
            method: "".concat(MODULE_NAME$1, "_chainID")
        },
        getGasPriceAdvice: {
            method: "".concat(MODULE_NAME$1, "_gasPriceAdvice"),
            outputFormatter: strToBigNumber
        },
        getNodeVersion: {
            method: "".concat(MODULE_NAME$1, "_nodeVersion")
        },
        // not necessary to write requester parameter
        getSdkVersion: function getSdkVersion() {
            return Promise.resolve("0.9.0");
        },
        watchBlock: function watchBlock(requester, withTxList, callback) {
            return requester.watch("".concat(MODULE_NAME$1, "_latestStableBlock"), [withTxList], callback);
        }
    };
    var apis$1 = Object.entries(apiList$1).map(function (_ref) {
        var _ref2 = slicedToArray(_ref, 2),
            key = _ref2[0],
            value = _ref2[1];

        if (typeof value === 'function') {
            return {
                name: key,
                call: value
            };
        }

        return objectSpread({
            name: key
        }, value);
    });
    var chain = {
        moduleName: MODULE_NAME$1,
        apis: apis$1
    };

    var MODULE_NAME$2 = 'mine';
    var apiList$2 = {
        getMining: {
            method: "".concat(MODULE_NAME$2, "_isMining")
        },
        getLemoBase: {
            method: "".concat(MODULE_NAME$2, "_lemoBase")
        }
    };
    var apis$2 = Object.entries(apiList$2).map(function (_ref) {
        var _ref2 = slicedToArray(_ref, 2),
            key = _ref2[0],
            value = _ref2[1];

        if (typeof value === 'function') {
            return {
                name: key,
                call: value
            };
        }

        return objectSpread({
            name: key
        }, value);
    });
    var mine = {
        moduleName: MODULE_NAME$2,
        apis: apis$2
    };

    var MODULE_NAME$3 = 'net';
    var apiList$3 = {
        getPeerCount: {
            method: "".concat(MODULE_NAME$3, "_getPeerCount")
        },
        getInfo: {
            method: "".concat(MODULE_NAME$3, "_info")
        }
    };
    var apis$3 = Object.entries(apiList$3).map(function (_ref) {
        var _ref2 = slicedToArray(_ref, 2),
            key = _ref2[0],
            value = _ref2[1];

        if (typeof value === 'function') {
            return {
                name: key,
                call: value
            };
        }

        return objectSpread({
            name: key
        }, value);
    });
    var net = {
        moduleName: MODULE_NAME$3,
        apis: apis$3
    };

    var MODULE_NAME$4 = 'tx';
    var apiList$4 = {
        sendTx: {
            method: "".concat(MODULE_NAME$4, "_sendTx")
        }
    };
    var apis$4 = Object.entries(apiList$4).map(function (_ref) {
        var _ref2 = slicedToArray(_ref, 2),
            key = _ref2[0],
            value = _ref2[1];

        if (typeof value === 'function') {
            return {
                name: key,
                call: value
            };
        }

        return objectSpread({
            name: key
        }, value);
    });
    var tx = {
        moduleName: MODULE_NAME$4,
        apis: apis$4
    };

    var dP$1 = _objectDp.f;
    var FProto = Function.prototype;
    var nameRE = /^\s*function ([^ (]*)/;
    var NAME$1 = 'name';

    // 19.2.4.2 name
    NAME$1 in FProto || _descriptors && dP$1(FProto, NAME$1, {
        configurable: true,
        get: function () {
            try {
                return ('' + this).match(nameRE)[1];
            } catch (e) {
                return '';
            }
        }
    });

    var Api =
        /*#__PURE__*/
        function () {
            function Api(config, requester) {
                classCallCheck(this, Api);

                this.name = config.name;
                this.inputFormatter = config.inputFormatter;
                this.outputFormatter = config.outputFormatter;
                this.call = config.call;
                this.method = config.method;
                this.requester = requester;
            }

            createClass(Api, [{
                key: "attachTo",
                value: function attachTo(obj) {
                    obj[this.name] = buildCall(this);
                }
            }]);

            return Api;
        }();

    function buildCall(api) {
        if (typeof api.call === 'function') {
            return api.call.bind(null, api.requester);
        }

        return (
            /*#__PURE__*/
            asyncToGenerator(
                /*#__PURE__*/
                regenerator.mark(function _callee() {
                    var _len,
                        args,
                        _key,
                        result,
                        _args = arguments;

                    return regenerator.wrap(function _callee$(_context) {
                        while (1) {
                            switch (_context.prev = _context.next) {
                                case 0:
                                    for (_len = _args.length, args = new Array(_len), _key = 0; _key < _len; _key++) {
                                        args[_key] = _args[_key];
                                    }

                                    if (api.inputFormatter) {
                                        args = api.inputFormatter(args);
                                    }

                                    _context.next = 4;
                                    return api.requester.send(api.method, args);

                                case 4:
                                    result = _context.sent;

                                    if (!api.outputFormatter) {
                                        _context.next = 7;
                                        break;
                                    }

                                    return _context.abrupt("return", api.outputFormatter(result));

                                case 7:
                                    return _context.abrupt("return", result);

                                case 8:
                                case "end":
                                    return _context.stop();
                            }
                        }
                    }, _callee, this);
                }))
        );
    }

    var LemoClient =
        /*#__PURE__*/
        function () {
            function LemoClient(config) {
                classCallCheck(this, LemoClient);

                // The Object.defineProperty is not work in otto. but we can name fields with first letter '_' to make it invisible
                this._requester = new Requester(newConn(config));

                this._createAPI(account.moduleName, account.apis);

                this._createAPI('', chain.apis); // attach the apis from chain to 'this'


                this._createAPI(mine.moduleName, mine.apis);

                this._createAPI(net.moduleName, net.apis);

                this._createAPI(tx.moduleName, tx.apis);
            }

            createClass(LemoClient, [{
                key: "_createAPI",
                value: function _createAPI(moduleName, apis) {
                    var _this = this;

                    if (moduleName && !this[moduleName]) {
                        this[moduleName] = {};
                    }

                    apis.forEach(function (api) {
                        new Api(api, _this._requester).attachTo(moduleName ? _this[moduleName] : _this);
                    });
                }
            }, {
                key: "stopWatch",
                value: function stopWatch(watchId) {
                    return this._requester.stopWatch(watchId);
                }
            }]);

            return LemoClient;
        }();

    function newConn(config) {
        if (!config) {
            return new HttpConn();
        }

        if (config) {
            // conn object
            if (typeof config.send === 'function') {
                return config;
            } // http conn config


            if (typeof config.host === 'string' && config.host.toLowerCase().startsWith('http')) {
                return new HttpConn(config.host, config.timeout, config.username, config.password, config.headers);
            }
        }

        throw new Error("unknown conn config: ".concat(config));
    }

    return LemoClient;

})));
