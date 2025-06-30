document.addEventListener('DOMContentLoaded', () => {
            const canvas = document.getElementById('spirographCanvas');
            const ctx = canvas.getContext('2d');
            const simBgColorPicker = document.getElementById('simBgColorPicker');
            const mainColorPickerEl = document.getElementById('mainColorPicker');
            const zoomSlider = document.getElementById('zoomSlider');
            const zoomValSpan = document.getElementById('zoom-val');
            const nodesConfigContainer = document.getElementById('nodes-config-container');
            const addNodeButton = document.getElementById('addNodeButton');
            const resetNodesConfigButton = document.getElementById('resetNodesConfigButton'); 
            const resetTracesButton = document.getElementById('resetTracesButton');       
            const startStopButton = document.getElementById('startStopButton');
            const downloadButton = document.getElementById('downloadButton');
            const globalSettingsGroup = document.getElementById('global-settings-group');
            const setupPanel = document.getElementById('setup-panel'); // Get setup panel
            const memoryStoreButton = document.getElementById('memoryStoreButton');
            const memoryRecallButton = document.getElementById('memoryRecallButton');
            const memoryButtonsRow = document.getElementById('memory-buttons-row');
            const appTitle = document.getElementById('app-title');
            const generateGifButton = document.getElementById('generateGifButton');


            let nodes = []; 
            let collapsedStates = {}; // To store collapsed states
            const MAX_NODES = 4; 
            const BASE_SPEED_RPM = 4.0; 
            const DT = 1/60; 
            const BASE_PHYSICS_SUB_STEPS = 5;

            let allTraceSegments = []; 
            let currentSegmentMap = new Map(); 
            let memorySlot = null; // For M+ / MR functionality

            let animationFrameId;
            let isRunning = false;
            let currentZoom = 1.0;
            let canvasOffsetX = 0; 
            let canvasOffsetY = 0; 
            let isPanning = false;
            let lastPanX, lastPanY;

            let initialPinchDistance = null; // For pinch zoom
            const ZOOM_SENSITIVITY = 0.001; // For mouse wheel zoom

            // Easter Egg State
            let pressTimer;
            let isSpinning = false;
            let spinAnimationId;
            let lastSpinFrameTime;
            let totalRotationAngle = 0;
            let currentSpinSpeed = 0; // RPM
            const maxSpinSpeed = 300; // RPM
            const accelerationDuration = 5000; // 5 seconds
            const decelerationDuration = 4000; // 4 seconds
            let spinStartTime;
            let decelStartTime;
            let initialSpinSpeedOnDecel;
            let isDecelerating = false;
            let isAligning = false;
            let startAngleOnAlign;
            let targetAngleOnAlign;
            let alignStartTime;
            let alignDuration;

            const themes = [
                { name: 'Cosmic Cyan', bg: '#1A1A2E', hl: '#53D8FB', nodeColors: ['#FF69B4', '#39FF14', '#FFD700', '#FF00FF'] },
                { name: 'Neon Magenta', bg: '#2C001E', hl: '#FF00A0', nodeColors: ['#00FFCD', '#FFDB00', '#0094FF', '#BFFF00'] },
                { name: 'Electric Lime', bg: '#0D2002', hl: '#A8FF00', nodeColors: ['#FF00E4', '#00E0FF', '#FF8400', '#E1FF00'] },
                { name: 'Golden Hour', bg: '#1F1F1F', hl: '#FFBF00', nodeColors: ['#FF3D3D', '#6A0DAD', '#3DFF3D', '#FF3DD0'] }
            ];
            let currentTheme;

            function hexToRgb(hex) { hex = hex.replace(/^#?([a-f\d])([a-f\d])([a-f\d])$/i, (m,r,g,b)=>r+r+g+g+b+b); const res = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex); return res?{r:parseInt(res[1],16),g:parseInt(res[2],16),b:parseInt(res[3],16)}:null; }
            function rgbToHex(r,g,b) { return "#"+((1<<24)+(r<<16)+(g<<8)+b).toString(16).slice(1).toUpperCase(); }
            function getLuminance(hex) { const rgb=hexToRgb(hex); return rgb?0.2126*rgb.r+0.7152*rgb.g+0.0722*rgb.b:0; }
            function adjustBrightness(hex,p) { const rgb=hexToRgb(hex); if(!rgb)return hex; let{r,g,b}=rgb; const a=Math.floor(255*(p/100)); r=Math.max(0,Math.min(255,r+a)); g=Math.max(0,Math.min(255,g+a)); b=Math.max(0,Math.min(255,b+a)); return rgbToHex(r,g,b); }
            function hexToRgba(hex,alpha) { const rgb=hexToRgb(hex); return rgb?`rgba(${rgb.r},${rgb.g},${rgb.b},${alpha})`:`rgba(0,0,0,${alpha})`; }
            function getContrastYIQ(hex){ const rgb=hexToRgb(hex); if(!rgb)return'white'; const yiq=((rgb.r*299)+(rgb.g*587)+(rgb.b*114))/1000; return(yiq>=128)?'black':'white'; }

            function updateSelectArrowColor() {
                const mainColor = getComputedStyle(document.documentElement).getPropertyValue('--main-color').trim();
                if (!mainColor) return; // Guard against running before styles are set
                const encodedColor = encodeURIComponent(mainColor.substring(1));
                const svgUrl = `url('data:image/svg+xml;charset=US-ASCII,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%22292.4%22%20height%3D%22292.4%22%3E%3Cpath%20fill%3D%22%23${encodedColor}%22%20d%3D%22M287%2069.4a17.6%2017.6%200%200%200-13-5.4H18.4c-5%200-9.3%201.8-12.9%205.4A17.6%2017.6%200%200%200%200%2082.2c0%205%201.8%209.3%205.4%2012.9l128%20127.9c3.6%203.6%207.8%205.4%2012.8%205.4s9.2-1.8%2012.8-5.4L287%2095c3.5-3.5%205.4-7.8%205.4-12.8%200-5-1.9-9.4-5.4-12.8z%22%2F%3E%3C%2Fsvg%3E')`;
                
                document.querySelectorAll('select').forEach(selectEl => {
                    selectEl.style.backgroundImage = svgUrl;
                });
            }

            function updateDynamicTheme() {
                const simBg = document.documentElement.style.getPropertyValue('--sim-background-color');
                const mainC = document.documentElement.style.getPropertyValue('--main-color');
                const lumSimBg = getLuminance(simBg);
                document.body.style.backgroundColor = adjustBrightness(simBg, lumSimBg < 128 ? 20 : -20);
                const panelBaseColor = adjustBrightness(simBg, lumSimBg < 128 ? 5 : -5);
                document.documentElement.style.setProperty('--panel-bg-color', hexToRgba(panelBaseColor, 0.85));
                document.documentElement.style.setProperty('--panel-border-color', adjustBrightness(panelBaseColor, lumSimBg < 128 ? 10 : -10));
                document.documentElement.style.setProperty('--input-bg-color', adjustBrightness(panelBaseColor, lumSimBg < 128 ? -5 : 5));
                document.documentElement.style.setProperty('--button-text-color', getContrastYIQ(mainC) === 'black' ? '#101020' : '#F0F0F8');
                const mainColorRGB = hexToRgb(mainC);
                if (mainColorRGB) document.documentElement.style.setProperty('--main-color-rgb', `${mainColorRGB.r},${mainColorRGB.g},${mainColorRGB.b}`);
                updateSelectArrowColor();
                document.querySelectorAll('input[type="range"]').forEach(updateSliderFill);
                if (!isRunning) drawStaticSpirograph();
            }

            function resizeCanvas() {
                const setupPanel = document.getElementById('setup-panel');
                const panelWidth = setupPanel.offsetWidth + 20;
                let availableWidth = window.innerWidth - 40;
                if (window.innerWidth > 768) availableWidth = window.innerWidth - panelWidth - 40 - 20;
                const availableHeight = window.innerHeight - 60;
                const simAreaElement = document.getElementById('simulation-area');
                const simAreaWidth = simAreaElement ? simAreaElement.clientWidth : availableWidth;
                availableWidth = Math.min(availableWidth, simAreaWidth);
                let canvasSize = Math.min(availableWidth, availableHeight, 800);
                canvas.width = canvas.height = Math.max(canvasSize, 300);
                if (!isRunning) drawStaticSpirograph(); else drawSpirographFrame(); 
            }

            function getLogValue(sliderValue, sMin, sMax, minV, maxV) {
                const minL = Math.log(minV); const maxL = Math.log(maxV);
                const scale = (maxL - minL) / (sMax - sMin);
                return Math.exp(minL + scale * (sliderValue - sMin));
            }
            
            function calculateAndSetZoom() {
                if (nodes.length === 0) {
                    if (!isRunning) drawStaticSpirograph();
                    return;
                }
                // Sum of all node lengths gives the maximum possible radius
                const maxRadius = nodes.reduce((sum, node) => sum + parseFloat(node.length), 0);
                const maxDiameter = maxRadius * 2;
                const canvasSize = canvas.width;
                let newZoom = 1.0; // Default zoom
                if (maxDiameter > 0) {
                    // Fit diameter within 95% of canvas size for a small margin
                    newZoom = (canvasSize * 0.95) / maxDiameter;
                }
                newZoom = Math.max(parseFloat(zoomSlider.min), Math.min(parseFloat(zoomSlider.max), newZoom));
                if (!isFinite(newZoom) || newZoom <= 0) newZoom = 1.0;
                currentZoom = newZoom;
                zoomSlider.value = currentZoom;
                zoomValSpan.textContent = currentZoom.toFixed(2);
                updateSliderFill(zoomSlider);
                console.log(`Auto-zoom calculated. Max radius: ${maxRadius.toFixed(0)}px, New zoom: ${currentZoom.toFixed(2)}x`);
                if (!isRunning) drawStaticSpirograph();
            }
            
            function resetPanAndRedraw() {
                canvasOffsetX = 0; canvasOffsetY = 0;
                if (!isRunning) drawStaticSpirograph();
            }
            
            function evaluateAngleExpressionToRadians(expressionStr, defaultDeg = -90) {
                let resultDeg = defaultDeg;
                try {
                    const sanitizedExpression = expressionStr
                        .replace(/PI/gi, Math.PI.toString()) 
                        .replace(/[^-()\d/*+%.MathpPIi\s]/g, ''); 
                    if (sanitizedExpression.trim() === "") resultDeg = defaultDeg;
                    else {
                        const evaluated = Function('"use strict";return (' + sanitizedExpression + ')')();
                        if (typeof evaluated === 'number' && isFinite(evaluated)) resultDeg = evaluated;
                        else console.warn(`Expression "${expressionStr}" evaluated to non-finite number: ${evaluated}. Using default.`);
                    }
                } catch (error) {
                    console.warn(`Error evaluating angle expression "${expressionStr}":`, error, `. Using default ${defaultDeg} deg.`);
                }
                return resultDeg * (Math.PI / 180); 
            }

            function evaluateRotationExpression(expressionStr, defaultVal = 0) {
                let result = defaultVal;
                try {
                    const sanitizedExpression = String(expressionStr || '')
                        .replace(/PI/gi, Math.PI.toString())
                        .replace(/[^-()\d/*+%.MathpPIi\s]/g, '');

                    if (sanitizedExpression.trim() === "") {
                        return defaultVal;
                    }

                    const evaluated = Function('"use strict";return (' + sanitizedExpression + ')')();

                    if (typeof evaluated === 'number' && isFinite(evaluated)) {
                        result = Math.max(0, evaluated); // Ensure non-negative
                    } else {
                        console.warn(`Expression "${expressionStr}" evaluated to non-finite number: ${evaluated}. Using default.`);
                    }
                } catch (error) {
                    console.warn(`Error evaluating rotation expression "${expressionStr}":`, error, `. Using default ${defaultVal}.`);
                }
                return result;
            }

            function updateNodeFromUI(nodeId, isAngleReset = false) {
                const node = nodes.find(n => n.id === nodeId); if (!node) return;
                let changedProperties = [];
                const lengthSlider = document.getElementById(`length${nodeId}`);
                if (lengthSlider) node.length = parseFloat(lengthSlider.value);
                const colorPicker = document.getElementById(`color${nodeId}`);
                if (colorPicker) node.color = colorPicker.value;
                const drawCheckbox = document.getElementById(`draw${nodeId}`);
                const widthSlider = document.getElementById(`width${nodeId}`);
                if (widthSlider) node.width = parseFloat(widthSlider.value);
                const alphaSlider = document.getElementById(`alpha${nodeId}`);
                if (alphaSlider) node.alpha = parseInt(alphaSlider.value);

                if (drawCheckbox) node.isDrawing = drawCheckbox.checked;

                if (node.id === 1) {
                    const speedSlider = document.getElementById(`speed${nodeId}`);
                    const directionSelect = document.getElementById(`direction${nodeId}`);
                    const startAngleInput = document.getElementById(`startAngle${nodeId}`);
                    const totalRotationsInput = document.getElementById(`totalRotations${nodeId}`);

                    if (speedSlider && directionSelect) {
                        const speedMultiplier = getLogValue(parseFloat(speedSlider.value), 0, 100, 0.1, 10);
                        node.speedMultiplier = speedMultiplier;
                        node.speed = BASE_SPEED_RPM * (2 * Math.PI / 60) * speedMultiplier;
                        node.direction = parseInt(directionSelect.value); 
                        changedProperties.push(`speed: ${speedMultiplier.toFixed(2)}x`, `direction: ${directionSelect.options[directionSelect.selectedIndex].text}`);
                        document.getElementById(`speed${nodeId}-val`).textContent = speedMultiplier.toFixed(2) + 'x';
                    }
                    if (startAngleInput && isAngleReset) { 
                        const defaultAngleDeg = (node.initialAbsoluteAngle * (180 / Math.PI));
                        node.initialAbsoluteAngle = evaluateAngleExpressionToRadians(startAngleInput.value, defaultAngleDeg);
                        node.currentAbsoluteAngle = node.initialAbsoluteAngle;
                        
                        // When resetting the base node, reset all children as well
                        for(let i = 1; i < nodes.length; i++) {
                            const childNode = nodes[i];
                            const parentNode = nodes[i-1];
                            childNode.currentRelativeAngle = childNode.initialRelativeAngle;
                            childNode.currentAbsoluteAngle = parentNode.currentAbsoluteAngle + childNode.currentRelativeAngle;
                        }
                        
                        changedProperties.push(`startAngle RESET to: ${startAngleInput.value} (evaluates to ${(node.initialAbsoluteAngle * 180/Math.PI).toFixed(2)}°)`);
                    }
                    if (totalRotationsInput) {
                        const expression = totalRotationsInput.value;
                        const finalValue = evaluateRotationExpression(expression, 0);
                        node.totalRotations = finalValue;
                        changedProperties.push(`totalRotations: ${expression} (evaluates to ${finalValue.toFixed(3)})`);
                    }
                } else {
                    const relSpeedInput = document.getElementById(`relSpeed${nodeId}`);
                    if (relSpeedInput) { node.relativeSpeed = parseFloat(relSpeedInput.value) || 0; changedProperties.push(`relativeSpeed: ${node.relativeSpeed}`); }
                }
                console.log(`Node ${nodeId} UI updated. Length: ${node.length}, Color: ${node.color}, Width: ${node.width}, Alpha: ${node.alpha}, Drawing: ${node.isDrawing}. ${changedProperties.join(', ')}`);
                
                if (!isRunning) {
                    if (isAngleReset) {
                         calculateAndSetZoom(); // This will redraw with the new static positions
                    } else {
                         drawStaticSpirograph(); // Just redraw for non-structural changes
                    }
                }
            }
            
            function calculateStaticNodePositions() { 
                let currentX = 0, currentY = 0;
                const positions = [];
                for (let i = 0; i < nodes.length; i++) {
                    const node = nodes[i];
                    
                    const armEndX = currentX + node.length * Math.cos(node.currentAbsoluteAngle);
                    const armEndY = currentY + node.length * Math.sin(node.currentAbsoluteAngle);
                    positions.push({
                        startX: currentX, startY: currentY, endX: armEndX, endY: armEndY,
                        nodeVisibleColor: node.isDrawing ? node.color : document.documentElement.style.getPropertyValue('--main-color'),
                        nodeWidth: node.width, nodeAlpha: node.alpha
                    });
                    currentX = armEndX; currentY = armEndY;
                }
                return positions;
            }

            function drawStaticSpirograph() {
                if (!ctx) return;
                ctx.fillStyle = document.documentElement.style.getPropertyValue('--sim-background-color');
                ctx.fillRect(0, 0, canvas.width, canvas.height);

                const canvasCenterX = canvas.width / 2 + canvasOffsetX; 
                const canvasCenterY = canvas.height / 2 + canvasOffsetY; 

                ctx.save();
                ctx.translate(canvasCenterX, canvasCenterY);
                ctx.rotate(totalRotationAngle);
                ctx.translate(-canvasCenterX, -canvasCenterY);

                const nodePositions = calculateStaticNodePositions();
                nodePositions.forEach(pos => {
                    ctx.beginPath();
                    ctx.moveTo(canvasCenterX + pos.startX * currentZoom, canvasCenterY + pos.startY * currentZoom);
                    ctx.lineTo(canvasCenterX + pos.endX * currentZoom, canvasCenterY + pos.endY * currentZoom);
                    ctx.strokeStyle = hexToRgba(pos.nodeVisibleColor, 0.5);
                    ctx.lineWidth = Math.max(1, pos.nodeWidth * currentZoom); ctx.stroke();
                    ctx.beginPath();
                    ctx.arc(canvasCenterX + pos.endX * currentZoom, canvasCenterY + pos.endY * currentZoom, Math.max(2, 5 * currentZoom), 0, 2 * Math.PI);
                    ctx.fillStyle = hexToRgba(pos.nodeVisibleColor, pos.nodeAlpha / 100); ctx.fill();
                });
                drawAllTraces();
                ctx.restore();
            }
            
            function drawAllTraces() {
                const canvasCenterX = canvas.width / 2 + canvasOffsetX; 
                const canvasCenterY = canvas.height / 2 + canvasOffsetY; 
                allTraceSegments.forEach(segment => { 
                    if (segment.points.length > 1) {
                        ctx.strokeStyle = hexToRgba(segment.color, segment.nodeAlpha / 100);
                        ctx.lineWidth = Math.max(1, segment.nodeWidth * currentZoom);
                        ctx.beginPath(); 
                        ctx.moveTo(canvasCenterX + segment.points[0].x * currentZoom, canvasCenterY + segment.points[0].y * currentZoom);
                        for (let k = 1; k < segment.points.length; k++) {
                            ctx.lineTo(canvasCenterX + segment.points[k].x * currentZoom, canvasCenterY + segment.points[k].y * currentZoom);
                        }
                        ctx.stroke();
                    }
                });
            }

            function drawSpirographFrame() {
                if (!isRunning) return;
                let currentPhysicsSubSteps = BASE_PHYSICS_SUB_STEPS;
                if (nodes.length === 3) currentPhysicsSubSteps = Math.round(BASE_PHYSICS_SUB_STEPS * 1.5);
                else if (nodes.length >= 4) currentPhysicsSubSteps = Math.round(BASE_PHYSICS_SUB_STEPS * 1.5 * 1.5);
                
                let haltSimulationAfterThisFrame = false;
                const dt_step = DT / currentPhysicsSubSteps;

                for (let step = 0; step < currentPhysicsSubSteps; step++) {
                    let currentLogicalX_step = 0, currentLogicalY_step = 0;
                    for (let i = 0; i < nodes.length; i++) {
                        const node = nodes[i];
                        let newAbsoluteAngleThisSubStep;
                        const prevAbsoluteAngleForSubStep = node.currentAbsoluteAngle; 
                        
                        if (i === 0) {
                            const targetAngle = node.totalRotations > 0 ? node.totalRotations * 2 * Math.PI : 0;
                            let dAngleBase = node.speed * dt_step;

                            // Check for overshoot and correct the final step to be precise
                            if (targetAngle > 0 && (node.totalAngleTraversed + dAngleBase >= targetAngle)) {
                                dAngleBase = targetAngle - node.totalAngleTraversed;
                                if (dAngleBase < 0) dAngleBase = 0; // Safeguard against moving backward
                                haltSimulationAfterThisFrame = true;
                            }

                            node.totalAngleTraversed += dAngleBase; // Always accumulate positive angle

                            if (node.direction === 0) { 
                                newAbsoluteAngleThisSubStep = node.initialAbsoluteAngle; 
                                node.virtual_dAngle = dAngleBase; 
                            } else {
                                const dAngle = dAngleBase * node.direction;
                                newAbsoluteAngleThisSubStep = prevAbsoluteAngleForSubStep + dAngle; 
                                node.virtual_dAngle = dAngle; 
                            }
                        } else {
                            const parentNode = nodes[i-1];
                            const dParentAbsoluteAngle = (parentNode.id === 1 && parentNode.direction === 0) ? 
                                                         parentNode.virtual_dAngle : 
                                                         (parentNode.currentAbsoluteAngle - parentNode.previousAbsoluteAngleSubStep);
                            
                            const dRelativeAngle = node.relativeSpeed * dParentAbsoluteAngle;
                            node.currentRelativeAngle += dRelativeAngle;
                            newAbsoluteAngleThisSubStep = parentNode.currentAbsoluteAngle + node.currentRelativeAngle;
                        }
                        node.previousAbsoluteAngleSubStep = node.currentAbsoluteAngle; 
                        node.currentAbsoluteAngle = newAbsoluteAngleThisSubStep; 

                        const armEndX_step = currentLogicalX_step + node.length * Math.cos(node.currentAbsoluteAngle);
                        const armEndY_step = currentLogicalY_step + node.length * Math.sin(node.currentAbsoluteAngle);

                        if (node.isDrawing) {
                            const activeSegment = currentSegmentMap.get(node.id);
                            if (activeSegment) activeSegment.points.push({ x: armEndX_step, y: armEndY_step });
                        }
                        currentLogicalX_step = armEndX_step; currentLogicalY_step = armEndY_step;
                    }
                    if (haltSimulationAfterThisFrame) {
                        break; // Exit sub-step loop after the final precise step
                    }
                }
                ctx.fillStyle = document.documentElement.style.getPropertyValue('--sim-background-color');
                ctx.fillRect(0, 0, canvas.width, canvas.height);

                const canvasCenterX = canvas.width / 2 + canvasOffsetX; 
                const canvasCenterY = canvas.height / 2 + canvasOffsetY; 

                ctx.save();
                ctx.translate(canvasCenterX, canvasCenterY);
                ctx.rotate(totalRotationAngle);
                ctx.translate(-canvasCenterX, -canvasCenterY);

                let currentLogicalX_draw = 0, currentLogicalY_draw = 0;

                for (let i = 0; i < nodes.length; i++) {
                    const node = nodes[i];
                    const armEndX_draw = currentLogicalX_draw + node.length * Math.cos(node.currentAbsoluteAngle);
                    const armEndY_draw = currentLogicalY_draw + node.length * Math.sin(node.currentAbsoluteAngle);
                    const displayColor = node.isDrawing ? node.color : document.documentElement.style.getPropertyValue('--main-color');

                    ctx.beginPath();
                    ctx.moveTo(canvasCenterX + currentLogicalX_draw * currentZoom, canvasCenterY + currentLogicalY_draw * currentZoom);
                    ctx.lineTo(canvasCenterX + armEndX_draw * currentZoom, canvasCenterY + armEndY_draw * currentZoom);
                    ctx.strokeStyle = hexToRgba(displayColor, 0.5);
                    ctx.lineWidth = Math.max(1, node.width * currentZoom); ctx.stroke();

                    ctx.beginPath();
                    ctx.arc(canvasCenterX + armEndX_draw * currentZoom, canvasCenterY + armEndY_draw * currentZoom, Math.max(2, 5 * currentZoom), 0, 2 * Math.PI);
                    ctx.fillStyle = hexToRgba(displayColor, node.alpha / 100); ctx.fill();

                    currentLogicalX_draw = armEndX_draw; currentLogicalY_draw = armEndY_draw;
                }
                drawAllTraces(); 
                ctx.restore();

                if (haltSimulationAfterThisFrame) {
                    const node1 = nodes[0];
                    console.log(`Target rotations (${node1.totalRotations}) reached. Halting simulation.`);
                    stopSimulation();
                } else {
                    animationFrameId = requestAnimationFrame(drawSpirographFrame);
                }
            }

            function createNodeControls(nodeId, isFirstNode) {
                const node = nodes.find(n => n.id === nodeId); if (!node) return null;
                const nodeDiv = document.createElement('div');
                nodeDiv.className = 'node-config'; nodeDiv.id = `node-config-${nodeId}`;
                if (collapsedStates[nodeId]) {
                    nodeDiv.classList.add('collapsed');
                }
                nodeDiv.setAttribute('data-runtime-hide', 'true');
                const removeBtnHtml = nodeId > 1 ? `<button class="remove-node-btn" data-nodeid="${nodeId}">Remove</button>` : '';
                
                let controlHtml = '';
                if (isFirstNode) {
                    const initialAngleDeg = (node.initialAbsoluteAngle * (180 / Math.PI)).toFixed(1);
                    controlHtml = `
                        <div class="input-container" id="node${nodeId}-start-angle-control">
                            <label for="startAngle${nodeId}">Starting Angle:</label>
                            <div class="input-group">
                                <input type="text" id="startAngle${nodeId}" value="${initialAngleDeg}" data-nodeid="${nodeId}" placeholder="e.g. -90 or 360/3">
                                <button id="setStartAngle${nodeId}" class="button-secondary">Set</button>
                            </div>
                        </div>
                        <div class="input-container" id="node${nodeId}-rotations-control">
                            <label for="totalRotations${nodeId}">Total Rotations (0 to run till stopped):</label>
                            <input type="text" id="totalRotations${nodeId}" value="${node.totalRotations}" data-nodeid="${nodeId}" placeholder="e.g. 10 or 2*PI">
                        </div>`;
                }

                nodeDiv.innerHTML = `
                    <h4 class="node-header">
                        <div class="node-title-group">
                            <span class="collapse-indicator">▼</span>
                            <span>Node ${nodeId}</span>
                        </div>
                        ${removeBtnHtml}
                    </h4>
                    <div class="node-controls">
                    ${controlHtml} 
                    <div class="slider-container" id="node${nodeId}-length-control">
                        <label for="length${nodeId}">Length: <span id="length${nodeId}-val">${node.length}</span>px</label>
                        <input type="range" id="length${nodeId}" min="10" max="${isFirstNode ? '1000' : '250'}" value="${node.length}" data-nodeid="${nodeId}">
                    </div>
                    <div class="color-picker-container" id="node${nodeId}-color-control">
                        <label for="color${nodeId}">Node Color (for trace):</label>
                        <input type="color" id="color${nodeId}" value="${node.color}" data-nodeid="${nodeId}">
                    </div>
                    <div class="slider-container" id="node${nodeId}-width-control">
                        <label for="width${nodeId}">Width: <span id="width${nodeId}-val">${node.width.toFixed(1)}</span>px</label>
                        <input type="range" id="width${nodeId}" min="1" max="10" value="${node.width}" step="0.1" data-nodeid="${nodeId}">
                    </div>
                    <div class="slider-container" id="node${nodeId}-alpha-control">
                        <label for="alpha${nodeId}">Alpha: <span id="alpha${nodeId}-val">${node.alpha}</span>%</label>
                        <input type="range" id="alpha${nodeId}" min="10" max="100" value="${node.alpha}" step="1" data-nodeid="${nodeId}">
                    </div>
                    ${isFirstNode ? `
                        <div class="slider-container" id="node1-speed-control-slider">
                            <label for="speed${nodeId}">Speed: <span id="speed${nodeId}-val">${node.speedMultiplier.toFixed(2)}x</span></label>
                            <input type="range" id="speed${nodeId}" min="0" max="100" value="50" data-nodeid="${nodeId}">
                        </div>
                        <div class="select-container" id="node1-speed-control-direction">
                            <label for="direction${nodeId}">Direction:</label>
                            <select id="direction${nodeId}" data-nodeid="${nodeId}">
                                <option value="1" ${node.direction === 1 ? 'selected' : ''}>Clockwise</option> 
                                <option value="-1" ${node.direction === -1 ? 'selected' : ''}>Anti-Clockwise</option> 
                                <option value="0" ${node.direction === 0 ? 'selected' : ''}>Fixed</option> 
                            </select>
                        </div>
                    ` : `
                        <div class="input-container" id="node${nodeId}-relspeed-control">
                            <label for="relSpeed${nodeId}">Relative Speed (to parent revolutions):</label>
                            <input type="number" id="relSpeed${nodeId}" value="${node.relativeSpeed}" step="0.1" data-nodeid="${nodeId}">
                        </div>
                    `}
                    <div class="checkbox-container" id="node${nodeId}-draw-control">
                        <input type="checkbox" id="draw${nodeId}" ${node.isDrawing ? 'checked' : ''} data-nodeid="${nodeId}">
                        <label for="draw${nodeId}">Enable Drawing</label>
                    </div>
                    </div>`;

                nodeDiv.querySelector('.node-header').addEventListener('click', (e) => {
                    if (e.target.classList.contains('remove-node-btn')) {
                        return;
                    }
                    nodeDiv.classList.toggle('collapsed');
                    collapsedStates[nodeId] = nodeDiv.classList.contains('collapsed');
                });

                const lengthSliderEl = nodeDiv.querySelector(`#length${nodeId}`);
                if (lengthSliderEl) {
                    lengthSliderEl.addEventListener('input', (e) => {
                        document.getElementById(`length${nodeId}-val`).textContent = e.target.value;
                        updateNodeFromUI(nodeId);
                        calculateAndSetZoom();
                        updateSliderFill(e.target); 
                    });
                    updateSliderFill(lengthSliderEl); 
                }
                const widthSliderEl = nodeDiv.querySelector(`#width${nodeId}`);
                if (widthSliderEl) {
                    widthSliderEl.addEventListener('input', (e) => {
                        document.getElementById(`width${nodeId}-val`).textContent = parseFloat(e.target.value).toFixed(1);
                        updateNodeFromUI(nodeId); updateSliderFill(e.target); });
                    updateSliderFill(widthSliderEl);
                }
                const alphaSliderEl = nodeDiv.querySelector(`#alpha${nodeId}`);
                if (alphaSliderEl) {
                    alphaSliderEl.addEventListener('input', (e) => {
                        document.getElementById(`alpha${nodeId}-val`).textContent = e.target.value;
                        updateNodeFromUI(nodeId); updateSliderFill(e.target); });
                    updateSliderFill(alphaSliderEl);
                }


                nodeDiv.querySelector(`#color${nodeId}`).addEventListener('input', () => updateNodeFromUI(nodeId));
                nodeDiv.querySelector(`#draw${nodeId}`).addEventListener('change', () => updateNodeFromUI(nodeId));
                if (isFirstNode) {
                    const speedSliderEl = nodeDiv.querySelector(`#speed${nodeId}`);
                    if (speedSliderEl) {
                         speedSliderEl.addEventListener('input', () => { updateNodeFromUI(nodeId); updateSliderFill(speedSliderEl); });
                        updateSliderFill(speedSliderEl); 
                    }
                    nodeDiv.querySelector(`#direction${nodeId}`).addEventListener('change', () => updateNodeFromUI(nodeId));
                    const startAngleInputEl = nodeDiv.querySelector(`#startAngle${nodeId}`);
                    if (startAngleInputEl) { 
                        startAngleInputEl.addEventListener('change', () => updateNodeFromUI(nodeId, false)); 
                    }
                    const setStartAngleBtn = nodeDiv.querySelector(`#setStartAngle${nodeId}`);
                    if(setStartAngleBtn) { 
                        setStartAngleBtn.addEventListener('click', () => updateNodeFromUI(nodeId, true));
                    }
                    const totalRotationsInputEl = document.getElementById(`totalRotations${nodeId}`);
                    if (totalRotationsInputEl) {
                        totalRotationsInputEl.addEventListener('change', () => updateNodeFromUI(nodeId, false));
                    }
                } else {
                    nodeDiv.querySelector(`#relSpeed${nodeId}`).addEventListener('input', () => updateNodeFromUI(nodeId));
                    if(removeBtnHtml) nodeDiv.querySelector('.remove-node-btn').addEventListener('click', () => removeNodeById(nodeId));
                }
                return nodeDiv;
            }

            function addNode() {
                if (nodes.length >= MAX_NODES) { alert("Maximum nodes reached."); return; }
                resetPanAndRedraw(); 
                const nodeId = nodes.length > 0 ? Math.max(...nodes.map(n => n.id)) + 1 : 1;
                const isFirstNode = (nodes.length === 0);
                const isSecondNode = (nodes.length === 1);
                const newNodeColor = currentTheme.nodeColors[nodes.length % currentTheme.nodeColors.length];

                let nodeLength = isFirstNode ? 100 : 50;
                let nodeIsDrawing = true;
                let nodeRelativeSpeed = isSecondNode ? 2 : 1;

                if (isFirstNode) {
                    nodeLength = Math.floor(Math.random() * (200 - 25 + 1)) + 25; // Random length 25-200
                    nodeIsDrawing = false; // Drawing disabled for Node 1
                } else if (isSecondNode) {
                    nodeLength = Math.floor(Math.random() * (150 - 25 + 1)) + 25; // Random length 25-150
                    const randomInt = Math.floor(Math.random() * 13) - 6; // -6 to +6
                    const randomFracOptions = [0, 0.05, 0.1, 0.2, 0.25, 0.5];
                    const randomFrac = randomFracOptions[Math.floor(Math.random() * randomFracOptions.length)];
                    nodeRelativeSpeed = randomInt + randomFrac;

                    const node1 = nodes[0];
                    if (node1) {
                        switch (randomFrac) {
                            case 0:    node1.totalRotations = 1; break;
                            case 0.05: node1.totalRotations = 20; break;
                            case 0.1:  node1.totalRotations = 10; break;
                            case 0.2:  node1.totalRotations = 5; break;
                            case 0.25: node1.totalRotations = 4; break;
                            case 0.5:  node1.totalRotations = 2; break;
                            default:   node1.totalRotations = 0; // Default case
                        }
                        const totalRotationsInput = document.getElementById(`totalRotations${node1.id}`);
                        if (totalRotationsInput) {
                            totalRotationsInput.value = node1.totalRotations;
                        }
                    }
                }

                const newNode = {
                    id: nodeId, length: nodeLength, color: newNodeColor, 
                    width: 1.5,
                    alpha: 100,
                    isDrawing: nodeIsDrawing,
                    totalRotations: 0,
                    totalAngleTraversed: 0,
                    speedMultiplier: 1.0, speed: BASE_SPEED_RPM * (2 * Math.PI / 60), 
                    direction: 1, 
                    relativeSpeed: nodeRelativeSpeed, 
                    currentAbsoluteAngle: 0, previousAbsoluteAngle: 0, currentRelativeAngle: 0, 
                    initialAbsoluteAngle: isFirstNode ? (-90 * Math.PI / 180) : 0, 
                    initialRelativeAngle: 0,
                    previousAbsoluteAngleSubStep: 0,
                    virtual_dAngle: 0, 
                };

                if (isFirstNode) {
                    newNode.currentAbsoluteAngle = newNode.initialAbsoluteAngle;
                } else {
                    const parentNode = nodes[nodes.length - 1];
                    // A new node should extend from the PARENT's CURRENT position.
                    // Its own initial relative angle is 0.
                    newNode.initialRelativeAngle = 0;
                    // Its initial absolute angle is thus the parent's current absolute angle.
                    newNode.initialAbsoluteAngle = parentNode.currentAbsoluteAngle + newNode.initialRelativeAngle;
                    newNode.currentAbsoluteAngle = newNode.initialAbsoluteAngle;
                }
                newNode.previousAbsoluteAngle = newNode.currentAbsoluteAngle;
                newNode.previousAbsoluteAngleSubStep = newNode.currentAbsoluteAngle;
                
                nodes.push(newNode);
                nodes.sort((a,b) => a.id - b.id); 
                const nodeControlsEl = createNodeControls(nodeId, isFirstNode);
                if (nodeControlsEl) {
                     nodesConfigContainer.appendChild(nodeControlsEl);
                     nodeControlsEl.querySelectorAll('input[type="range"]').forEach(updateSliderFill);
                     console.log(`Node ${nodeId} added. Initial length: ${newNode.length}, color: ${newNode.color}, drawing: ${newNode.isDrawing}`);
                }
                if (isFirstNode && document.getElementById(`speed${nodeId}`)) {
                     const speedSlider = document.getElementById(`speed${nodeId}`);
                     speedSlider.value = 50; updateSliderFill(speedSlider); 
                     document.getElementById(`speed${nodeId}-val`).textContent = newNode.speedMultiplier.toFixed(2) + 'x';
                }
                updateAddRemoveButtons();
                calculateAndSetZoom();
            }
            
            function removeNodeById(nodeIdToRemove) {
                if (nodes.length <= 1 || nodeIdToRemove === 1) return; 
                resetPanAndRedraw(); 
                nodes = nodes.filter(n => n.id !== nodeIdToRemove);
                const nodeConfigEl = document.getElementById(`node-config-${nodeIdToRemove}`);
                if (nodeConfigEl) nodeConfigEl.remove();
                console.log(`Node ${nodeIdToRemove} removed.`);
                updateAddRemoveButtons();
                calculateAndSetZoom();
            }

            function resetNodeConfigurations() { 
                if (isRunning) stopSimulation();
                nodes = []; 
                collapsedStates = {}; // Reset collapsed states
                nodesConfigContainer.innerHTML = ''; 
                addNode(); addNode(); 
                updateAddRemoveButtons();
                if (!allTraceSegments.some(seg => seg.points.length > 0)) {
                    console.log("Node configurations reset. No traces present.");
                    downloadButton.classList.add('hidden');
                } else {
                    downloadButton.classList.remove('hidden');
                }
                updateSelectArrowColor();
                if (!isRunning) drawStaticSpirograph(); 
            }

            function clearAllDrawingTraces() { 
                allTraceSegments = [];
                currentSegmentMap.clear();
                downloadButton.classList.add('hidden');
                console.log("All drawing traces cleared.");
                if (!isRunning) drawStaticSpirograph(); 
                else { 
                    ctx.fillStyle = document.documentElement.style.getPropertyValue('--sim-background-color');
                    ctx.fillRect(0, 0, canvas.width, canvas.height);
                }
            }
            
            function updateAddRemoveButtons() { addNodeButton.disabled = nodes.length >= MAX_NODES; }

            function updateSliderFill(slider) {
                if (!slider) return;
                const min = parseFloat(slider.min) || 0; const max = parseFloat(slider.max) || 100;
                const val = parseFloat(slider.value) || 0;
                const percentage = ((val - min) / (max - min)) * 100;
                slider.style.background = `linear-gradient(to right, 
                    var(--slider-track-fill-color) 0%, var(--slider-track-fill-color) ${percentage}%, 
                    var(--slider-track-bg-color) ${percentage}%, var(--slider-track-bg-color) 100%)`;
            }

            function startSimulation() {
                isRunning = true;
                startStopButton.textContent = 'Stop Simulation';
                setControlsVisibility(true);
                currentSegmentMap.clear();

                // Update totalRotations from UI just before starting
                const node1 = nodes[0];
                if (node1) {
                    const totalRotationsInput = document.getElementById(`totalRotations${node1.id}`);
                    if (totalRotationsInput) {
                        const expression = totalRotationsInput.value;
                        const finalValue = evaluateRotationExpression(expression, 0);
                        node1.totalRotations = finalValue;
                    }
                    node1.totalAngleTraversed = 0; // Reset rotation counter for the new run
                }

                nodes.forEach(node => {
                    // Angles are NOT reset here. They persist from the previous state.
                    node.previousAbsoluteAngle = node.currentAbsoluteAngle;
                    node.previousAbsoluteAngleSubStep = node.currentAbsoluteAngle;

                    if (node.isDrawing) {
                        const newSegment = {
                            nodeId: node.id,
                            color: node.color,
                            points: [],
                            nodeWidth: node.width,
                            nodeAlpha: node.alpha
                        };

                        // To correctly start the new trace segment, find the node's current end point.
                        let endX = 0;
                        let endY = 0;
                        for (let i = 0; i < nodes.length; i++) {
                            const n = nodes[i];
                            endX += n.length * Math.cos(n.currentAbsoluteAngle);
                            endY += n.length * Math.sin(n.currentAbsoluteAngle);
                            if (n.id === node.id) {
                                break; // Found our node
                            }
                        }
                        newSegment.points.push({ x: endX, y: endY });

                        allTraceSegments.push(newSegment);
                        currentSegmentMap.set(node.id, newSegment);
                    }
                });
                animationFrameId = requestAnimationFrame(drawSpirographFrame);
                console.log("Simulation started, resuming from current angles.");
            }

            function stopSimulation() {
                isRunning = false; if (animationFrameId) cancelAnimationFrame(animationFrameId);
                animationFrameId = null; startStopButton.textContent = 'Start Simulation';
                setControlsVisibility(false); 
                currentSegmentMap.clear(); 
                console.log("Simulation stopped.");
                if (allTraceSegments.some(seg => seg.points.length > 0)) downloadButton.classList.remove('hidden');
            }
            
            function setControlsVisibility(simRunning) {
                globalSettingsGroup.classList.toggle('control-hidden-during-run', simRunning);
                document.querySelectorAll('.node-config').forEach(el => {
                    const nodeId = parseInt(el.id.split('-').pop());
                    const header = el.querySelector('h4');

                    if (nodeId === 1) { 
                        if (header) {
                            header.classList.toggle('sim-running', simRunning);
                        }
                        // When running, we want to see speed/direction, so force-expand it visually
                        if (simRunning) {
                            el.classList.remove('collapsed');
                        } else {
                            // When stopping, restore its actual collapsed state
                            if (collapsedStates[nodeId]) {
                                el.classList.add('collapsed');
                            }
                        }

                        Array.from(el.querySelectorAll('.node-controls > div')).forEach(child => {
                            const isSpeedControl = child.id === 'node1-speed-control-slider' || child.id === 'node1-speed-control-direction';
                            
                            if (simRunning) { 
                                if (!isSpeedControl) {
                                    child.classList.add('control-hidden-during-run');
                                } else {
                                    child.classList.remove('control-hidden-during-run');
                                }
                                
                                const startAngleInput = el.querySelector(`#startAngle1`);
                                if(startAngleInput) startAngleInput.disabled = true;
                                const setAngleBtn = el.querySelector(`#setStartAngle1`);
                                if(setAngleBtn) setAngleBtn.classList.add('control-hidden-during-run');
                                const totalRotationsInput = el.querySelector(`#totalRotations1`);
                                if(totalRotationsInput) totalRotationsInput.disabled = true;

                            } else { 
                                child.classList.remove('control-hidden-during-run'); 
                                const startAngleInput = el.querySelector(`#startAngle1`);
                                if(startAngleInput) startAngleInput.disabled = false;
                                const setAngleBtn = el.querySelector(`#setStartAngle1`);
                                if(setAngleBtn) setAngleBtn.classList.remove('control-hidden-during-run');
                                const totalRotationsInput = el.querySelector(`#totalRotations1`);
                                if(totalRotationsInput) totalRotationsInput.disabled = false;
                            }
                        });
                    } else { 
                         el.classList.toggle('control-hidden-during-run', simRunning);
                    }
                });
                addNodeButton.classList.toggle('control-hidden-during-run', simRunning);
                resetNodesConfigButton.classList.toggle('control-hidden-during-run', simRunning); 
                resetTracesButton.classList.toggle('control-hidden-during-run', simRunning);    
                memoryButtonsRow.classList.toggle('control-hidden-during-run', simRunning);


                if (simRunning) {
                    downloadButton.classList.add('control-hidden-during-run');
                    downloadButton.classList.add('hidden');
                    generateGifButton.classList.add('control-hidden-during-run');
                    generateGifButton.classList.add('hidden');
                    appTitle.style.pointerEvents = 'none';
                } else {
                    downloadButton.classList.remove('control-hidden-during-run');
                    generateGifButton.classList.remove('control-hidden-during-run');
                     if (!allTraceSegments.some(seg => seg.points.length > 0)) {
                        downloadButton.classList.add('hidden');
                        generateGifButton.classList.add('hidden');
                     }
                     else {
                        downloadButton.classList.remove('hidden');
                        generateGifButton.classList.remove('hidden');
                     }
                    appTitle.style.pointerEvents = 'auto';
                }
            }

            // --- Panning & Touch Logic ---
            function getPointerPosition(event) {
                if (event.touches && event.touches.length > 0) {
                    return { x: event.touches[0].clientX, y: event.touches[0].clientY };
                }
                return { x: event.clientX, y: event.clientY };
            }

            function onPanStart(event) {
                isPanning = true;
                const pos = getPointerPosition(event);
                lastPanX = pos.x; lastPanY = pos.y;
                canvas.classList.add('grabbing');

                if (event.touches && event.touches.length === 2) {
                    isPanning = false; // Prioritize pinch zoom over pan if two fingers
                    initialPinchDistance = getPinchDistance(event);
                } else if (event.type === 'touchstart') {
                    event.preventDefault(); 
                }
            }

            function onPanMove(event) {
                if (event.touches && event.touches.length === 2) {
                    handlePinchZoom(event);
                    return;
                }
                if (!isPanning || (event.touches && event.touches.length > 1)) return;
                if (event.type === 'touchmove') event.preventDefault();

                const pos = getPointerPosition(event);
                const dx = pos.x - lastPanX;
                const dy = pos.y - lastPanY;
                canvasOffsetX += dx;
                canvasOffsetY += dy;
                lastPanX = pos.x;
                lastPanY = pos.y;
                if (!isRunning) drawStaticSpirograph();
            }

            function onPanEnd() {
                isPanning = false;
                canvas.classList.remove('grabbing');
                initialPinchDistance = null; // Reset pinch distance
                console.log(`Pan ended. Final OffsetX: ${canvasOffsetX.toFixed(2)}, OffsetY: ${canvasOffsetY.toFixed(2)}`);
            }

            function getPinchDistance(event) {
                const t1 = event.touches[0];
                const t2 = event.touches[1];
                return Math.sqrt(Math.pow(t2.clientX - t1.clientX, 2) + Math.pow(t2.clientY - t1.clientY, 2));
            }

            function handlePinchZoom(event) {
                if (!initialPinchDistance || event.touches.length !== 2) return;
                event.preventDefault();
                const newPinchDistance = getPinchDistance(event);
                const zoomFactor = newPinchDistance / initialPinchDistance;

                currentZoom *= zoomFactor;
                currentZoom = Math.max(parseFloat(zoomSlider.min), Math.min(parseFloat(zoomSlider.max), currentZoom));
                
                zoomSlider.value = currentZoom;
                zoomValSpan.textContent = currentZoom.toFixed(2);
                updateSliderFill(zoomSlider);
                
                initialPinchDistance = newPinchDistance; // Update for continuous zoom
                console.log(`Canvas pinch zoomed. New zoom: ${currentZoom.toFixed(2)}`);
                if (!isRunning) drawStaticSpirograph();
            }

            function handleMouseWheelZoom(event) {
                event.preventDefault(); // Prevent page scrolling
                const delta = event.deltaY * ZOOM_SENSITIVITY * -1; // Invert scroll direction for intuitive zoom
                currentZoom += delta;
                currentZoom = Math.max(parseFloat(zoomSlider.min), Math.min(parseFloat(zoomSlider.max), currentZoom));

                zoomSlider.value = currentZoom;
                zoomValSpan.textContent = currentZoom.toFixed(2);
                updateSliderFill(zoomSlider);
                console.log(`Canvas mouse wheel zoomed. New zoom: ${currentZoom.toFixed(2)}`);
                if (!isRunning) drawStaticSpirograph();
            }

            canvas.addEventListener('mousedown', onPanStart);
            canvas.addEventListener('mousemove', onPanMove);
            canvas.addEventListener('mouseup', onPanEnd);
            canvas.addEventListener('mouseleave', onPanEnd);
            canvas.addEventListener('touchstart', onPanStart, { passive: false });
            canvas.addEventListener('touchmove', onPanMove, { passive: false });
            canvas.addEventListener('touchend', onPanEnd);
            canvas.addEventListener('touchcancel', onPanEnd);
            canvas.addEventListener('wheel', handleMouseWheelZoom, { passive: false });


            simBgColorPicker.addEventListener('input', (e) => { 
                document.documentElement.style.setProperty('--sim-background-color', e.target.value); 
                console.log(`Global setting: Background color changed to ${e.target.value}`);
                updateDynamicTheme(); 
            });
            mainColorPickerEl.addEventListener('input', (e) => { 
                document.documentElement.style.setProperty('--main-color', e.target.value); 
                console.log(`Global setting: Highlight color changed to ${e.target.value}`);
                updateDynamicTheme(); 
            });
            zoomSlider.addEventListener('input', (e) => {
                currentZoom = parseFloat(e.target.value); 
                zoomValSpan.textContent = currentZoom.toFixed(2);
                console.log(`Global setting: Zoom level changed to ${currentZoom.toFixed(2)}x`);
                updateSliderFill(e.target); 
                resetPanAndRedraw(); 
            });
            addNodeButton.addEventListener('click', addNode); 
            resetNodesConfigButton.addEventListener('click', resetNodeConfigurations); 
            resetTracesButton.addEventListener('click', clearAllDrawingTraces);       
            startStopButton.addEventListener('click', () => { if (isRunning) stopSimulation(); else startSimulation(); }); 
            
            // --- Memory Button Logic ---
            memoryStoreButton.addEventListener('mousedown', handleMemoryButtonPressStart);
            memoryStoreButton.addEventListener('touchstart', handleMemoryButtonPressStart, { passive: false });
            memoryStoreButton.addEventListener('mouseup', handleMemoryButtonPressEnd);
            memoryStoreButton.addEventListener('touchend', handleMemoryButtonPressEnd);
            memoryStoreButton.addEventListener('mouseleave', handleMemoryButtonPressEnd); // Stop if mouse leaves button

            memoryRecallButton.addEventListener('click', () => {
                if (!memorySlot) return;
                if (isRunning) stopSimulation();

                currentZoom = memorySlot.zoom;
                canvasOffsetX = memorySlot.offsetX;
                canvasOffsetY = memorySlot.offsetY;
                simBgColorPicker.value = memorySlot.bgColor;
                mainColorPickerEl.value = memorySlot.hlColor;
                
                zoomSlider.value = currentZoom;
                updateSliderFill(zoomSlider);
                zoomValSpan.textContent = currentZoom.toFixed(2);

                document.documentElement.style.setProperty('--sim-background-color', memorySlot.bgColor);
                document.documentElement.style.setProperty('--main-color', memorySlot.hlColor);
                updateDynamicTheme();

                allTraceSegments = JSON.parse(JSON.stringify(memorySlot.traces));
                
                drawStaticSpirograph(); 
                console.log("State recalled from memory.");
            });


            downloadButton.addEventListener('click', () => {
                if (!allTraceSegments.some(seg => seg.points.length > 0)) { alert("No spirograph trace."); return; }
                const tempCanvas = document.createElement('canvas'); tempCanvas.width = canvas.width; tempCanvas.height = canvas.height;
                const tempCtx = tempCanvas.getContext('2d');
                tempCtx.fillStyle = document.documentElement.style.getPropertyValue('--sim-background-color');
                tempCtx.fillRect(0, 0, tempCanvas.width, tempCanvas.height);
                const cX = tempCanvas.width/2 + canvasOffsetX; 
                const cY = tempCanvas.height/2 + canvasOffsetY; 
                
                allTraceSegments.forEach(segment => { 
                    if (segment.points.length > 1) { 
                        tempCtx.strokeStyle = hexToRgba(segment.color, segment.nodeAlpha / 100);
                        tempCtx.lineWidth = Math.max(1, segment.nodeWidth * currentZoom);
                        tempCtx.beginPath();
                        tempCtx.moveTo(cX + segment.points[0].x * currentZoom, cY + segment.points[0].y * currentZoom);
                        for (let k = 1; k < segment.points.length; k++) tempCtx.lineTo(cX + segment.points[k].x * currentZoom, cY + segment.points[k].y * currentZoom);

                    tempCtx.stroke(); }});
                const link = document.createElement('a'); link.href = tempCanvas.toDataURL('image/png');
                link.download = 'spirograph_v2.6.png'; document.body.appendChild(link); link.click(); document.body.removeChild(link);
            });

            generateGifButton.addEventListener('click', generateGif);

            function generateGif() {
                console.log("Starting GIF generation with Animated_GIF...");
                if (!allTraceSegments.some(seg => seg.points.length > 0)) {
                    alert("No spirograph trace to generate a GIF from.");
                    console.log("GIF generation aborted: No trace data.");
                    return;
                }

                generateGifButton.disabled = true;
                generateGifButton.textContent = 'Generating GIF...';

                const ag = new Animated_GIF({
                    width: 420,
                    height: 420,
                    dithering: null,
                    palette: null,
                    delay: 20,
                    repeat: 0,
                    sampleInterval: 10,
                    numWorkers: 2
                });

                const totalFrames = 90; // 1.8s at 50fps
                const frameDelay = 20; // 20ms for 50fps
                ag.setDelay(frameDelay / 1000);

                const tempCanvas = document.createElement('canvas');
                const newResolution = 420;
                tempCanvas.width = newResolution;
                tempCanvas.height = newResolution;
                const tempCtx = tempCanvas.getContext('2d');

                const rotationPerFrame = (2 * Math.PI) / totalFrames;
                const direction = nodes[0].direction === 0 ? 1 : nodes[0].direction;

                console.log(`Generating ${totalFrames} frames for a ${newResolution}x${newResolution} GIF at 50 FPS...`);

                for (let i = 0; i < totalFrames; i++) {
                    const rotation = i * rotationPerFrame * direction;
                    drawGifFrame(tempCtx, rotation, newResolution);
                    ag.addFrame(tempCanvas);
                }

                ag.onRenderProgress(function(progress) {
                    generateGifButton.textContent = `Generating... ${Math.round(progress * 100)}%`;
                });

                ag.getBlobGIF(function(blob) {
                    console.log("GIF created successfully. Triggering download.");
                    const link = document.createElement('a');
                    link.href = URL.createObjectURL(blob);
                    link.download = 'spirograph.gif';
                    document.body.appendChild(link);
                    link.click();
                    document.body.removeChild(link);

                    generateGifButton.disabled = false;
                    generateGifButton.textContent = 'Generate GIF';
                    ag.destroy();
                });
            }

            function drawGifFrame(ctx, rotation, resolution) {
                ctx.fillStyle = document.documentElement.style.getPropertyValue('--sim-background-color');
                ctx.fillRect(0, 0, resolution, resolution);

                const scale = resolution / canvas.width;
                const canvasCenterX = (resolution / 2) + canvasOffsetX * scale;
                const canvasCenterY = (resolution / 2) + canvasOffsetY * scale;

                ctx.save();
                ctx.translate(canvasCenterX, canvasCenterY);
                ctx.rotate(rotation);
                ctx.translate(-canvasCenterX, -canvasCenterY);

                allTraceSegments.forEach(segment => {
                    if (segment.points.length > 1) {
                        ctx.strokeStyle = hexToRgba(segment.color, segment.nodeAlpha / 100);
                        ctx.lineWidth = Math.max(1, segment.nodeWidth * currentZoom * scale);
                        ctx.beginPath();
                        ctx.moveTo(canvasCenterX + segment.points[0].x * currentZoom * scale, canvasCenterY + segment.points[0].y * currentZoom * scale);
                        for (let k = 1; k < segment.points.length; k++) {
                            ctx.lineTo(canvasCenterX + segment.points[k].x * currentZoom * scale, canvasCenterY + segment.points[k].y * currentZoom * scale);
                        }
                        ctx.stroke();
                    }
                });
                ctx.restore();
            }

            // --- Easter Egg Logic ---

            function storeMemoryState() {
                if (isRunning) stopSimulation();
                memorySlot = {
                    traces: JSON.parse(JSON.stringify(allTraceSegments)),
                    zoom: currentZoom,
                    offsetX: canvasOffsetX,
                    offsetY: canvasOffsetY,
                    bgColor: simBgColorPicker.value,
                    hlColor: mainColorPickerEl.value
                };
                memoryRecallButton.disabled = false;
                console.log("State saved to memory.");
            }

            function handleMemoryButtonPressStart(e) {
                if (isRunning) return;
                e.preventDefault();

                // If already spinning, this press is an accelerator.
                if (isSpinning) {
                    startSpinning();
                    return;
                }

                // If not spinning, set a timer to initiate the spin on hold.
                clearTimeout(pressTimer); // Clear any previous stray timers
                pressTimer = setTimeout(() => {
                    pressTimer = null; // Timer has fired, nullify it
                    startSpinning();
                }, 1000);
            }

            function handleMemoryButtonPressEnd(e) {
                // If pressTimer is not null, it means the hold was released before 1s.
                // This is our "click" action.
                if (pressTimer) {
                    clearTimeout(pressTimer);
                    pressTimer = null;
                    // Execute original M+ button logic
                    storeMemoryState();
                } else if (isSpinning) {
                    // If there's no timer but we are spinning, it means the hold was long enough
                    // to start the spin, and now we need to stop it.
                    stopSpinning();
                }
            }

            function handlePressStart(e) {
                if (isRunning) return;
                e.preventDefault();
                clearTimeout(pressTimer);

                // If we are currently spinning down, re-accelerate immediately.
                if (isDecelerating) {
                    startSpinning();
                } else if (!isSpinning) { // Only set timer if not already accelerating
                    // Otherwise, wait for the 1-second hold to initiate the spin.
                    pressTimer = setTimeout(() => {
                        startSpinning();
                    }, 1000);
                }
            }

            function handlePressEnd() {
                clearTimeout(pressTimer);
                // Only stop if we are in the acceleration phase.
                if (isSpinning && !isDecelerating) {
                    stopSpinning();
                }
            }

            function startSpinning() {
                if (spinAnimationId) cancelAnimationFrame(spinAnimationId);

                const wasDecelerating = isDecelerating;
                isSpinning = true;
                isDecelerating = false;

                if (wasDecelerating) {
                    // We are re-accelerating. We need to find the "time" on the acceleration
                    // curve that corresponds to our current speed.
                    const currentSpeedRatio = currentSpinSpeed / maxSpinSpeed;
                    // Inverse of the log1p function
                    const progress = (Math.exp(currentSpeedRatio) - 1) / (Math.E - 1);
                    const equivalentTime = progress * accelerationDuration;
                    spinStartTime = performance.now() - equivalentTime;
                } else {
                    // Starting from zero.
                    currentSpinSpeed = 0;
                    spinStartTime = performance.now();
                }

                lastSpinFrameTime = performance.now();
                spinAnimationId = requestAnimationFrame(accelerateLoop);
            }

            function stopSpinning() {
                if (spinAnimationId) cancelAnimationFrame(spinAnimationId);

                isDecelerating = true;
                decelStartTime = performance.now();
                startAngleOnDecel = totalRotationAngle;
                initialSpinSpeedOnDecel = currentSpinSpeed;

                // Use a simple physics model (d = v*t + 0.5*a*t^2) to find a natural coasting distance.
                // For a smooth stop (v_final = 0), this simplifies to d = 0.5 * v_initial * t.
                const initialSpeedRadPerSec = (initialSpinSpeedOnDecel * 2 * Math.PI) / 60;
                const travelAngle = 0.5 * initialSpeedRadPerSec * (decelerationDuration / 1000);
                const preliminaryTarget = startAngleOnDecel + travelAngle;

                // Find the next full revolution *after* this natural stopping point.
                const targetRevolutions = Math.ceil(preliminaryTarget / (2 * Math.PI));
                targetAngleOnDecel = targetRevolutions * 2 * Math.PI;

                // Also enforce a minimum travel distance for a satisfying spin at low speeds.
                const minTargetAngle = startAngleOnDecel + (2 * Math.PI); // At least one full rotation.
                targetAngleOnDecel = Math.max(targetAngleOnDecel, minTargetAngle);

                lastSpinFrameTime = performance.now();
                spinAnimationId = requestAnimationFrame(decelerateLoop);
            }

            function accelerateLoop(currentTime) {
                if (!lastSpinFrameTime) lastSpinFrameTime = currentTime;
                const deltaTime = (currentTime - lastSpinFrameTime) / 1000;

                const timeSinceSpinStart = currentTime - spinStartTime;
                const progress = Math.min(timeSinceSpinStart / accelerationDuration, 1.0);
                currentSpinSpeed = maxSpinSpeed * Math.log1p(progress * (Math.E - 1));

                const rpm_rad_per_sec = currentSpinSpeed * 2 * Math.PI / 60;
                totalRotationAngle += rpm_rad_per_sec * deltaTime;

                drawStaticSpirograph();
                lastSpinFrameTime = currentTime;
                spinAnimationId = requestAnimationFrame(accelerateLoop);
            }

            function decelerateLoop(currentTime) {
                const timeSinceDecel = currentTime - decelStartTime;

                if (timeSinceDecel >= decelerationDuration) {
                    totalRotationAngle = 0; // Final reset for perfect alignment.
                    currentSpinSpeed = 0;
                    isSpinning = false;
                    isDecelerating = false;
                    spinAnimationId = null;
                } else {
                    // Use ease-out quad, which corresponds to constant deceleration.
                    const t = timeSinceDecel / decelerationDuration;
                    const easedT = t * (2 - t);
                    const newAngle = startAngleOnDecel + (targetAngleOnDecel - startAngleOnDecel) * easedT;

                    // Live-calculate speed for smooth re-acceleration
                    const deltaTime = (currentTime - lastSpinFrameTime) / 1000;
                    if (deltaTime > 0) {
                        const angleChange = newAngle - totalRotationAngle;
                        currentSpinSpeed = ((angleChange / deltaTime) * 60) / (2 * Math.PI);
                    }
                    totalRotationAngle = newAngle;
                }

                drawStaticSpirograph();
                lastSpinFrameTime = currentTime;

                if (isDecelerating) {
                    spinAnimationId = requestAnimationFrame(decelerateLoop);
                }
            }

            function initialize() {
                currentTheme = themes[Math.floor(Math.random() * themes.length)];
                document.documentElement.style.setProperty('--sim-background-color', currentTheme.bg);
                document.documentElement.style.setProperty('--main-color', currentTheme.hl);
                simBgColorPicker.value = currentTheme.bg; mainColorPickerEl.value = currentTheme.hl;
                updateDynamicTheme();

                // Set canvas size *before* adding nodes so initial auto-zoom is correct.
                resizeCanvas();
                window.addEventListener('resize', resizeCanvas);
                addNode(); addNode();

                // Move Global Settings group to its new position
                setupPanel.insertBefore(globalSettingsGroup, addNodeButton);

                setControlsVisibility(false); 
                updateAddRemoveButtons(); 
                updateSliderFill(zoomSlider);
                memoryRecallButton.disabled = true; // Initially disabled
                updateSelectArrowColor();

                // Easter Egg Listeners
                appTitle.addEventListener('mousedown', handlePressStart);
                appTitle.addEventListener('touchstart', handlePressStart, { passive: false });
                appTitle.addEventListener('mouseup', handlePressEnd);
                appTitle.addEventListener('mouseleave', handlePressEnd);
                appTitle.addEventListener('touchend', handlePressEnd);
                appTitle.addEventListener('touchcancel', handlePressEnd);
                
                if ('serviceWorker' in navigator) {
                    // This logic handles the auto-update flow.
                    // When a new service worker is activated, the page is reloaded
                    // to ensure the user gets the latest version of the app.
                    let refreshing;
                    navigator.serviceWorker.addEventListener('controllerchange', () => {
                        if (refreshing) return;
                        console.log('Controller changed, reloading for update...');
                        window.location.reload();
                        refreshing = true;
                    });

                    navigator.serviceWorker.register('sw.js')
                    .then((registration) => {
                        console.log('Service Worker registered for SpiroGen v2.5. Scope:', registration.scope);
                        
                        // Listen for the updatefound event.
                        registration.onupdatefound = () => {
                            const installingWorker = registration.installing;
                            if (installingWorker == null) {
                                return;
                            }
                            installingWorker.onstatechange = () => {
                                if (installingWorker.state === 'installed') {
                                    if (navigator.serviceWorker.controller) {
                                        // A new service worker has been installed.
                                        // Because sw.js calls self.skipWaiting(), it will
                                        // activate automatically, triggering the 'controllerchange'
                                        // event handled above.
                                        console.log('New service worker installed. Page will reload shortly.');
                                    } else {
                                        console.log('Content is cached for offline use.');
                                    }
                                }
                            };
                        };
                    })
                    .catch((error) => { console.error('Service Worker registration failed. Ensure sw.js is present and correctly configured. Error:', error); });
                }
            }
            initialize();
        });